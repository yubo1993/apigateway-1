#!/etc/bin/env python
# -*- coding: utf-8 -*-

"""
Node:
    /Node/Node-0123456/Name

"""

import sys
import uuid
import etcd3
import logging
import simplejson as json

from .const import *


class ClientNotExist(Exception):
    pass


class PredefinedTypeError(TypeError):
    pass


class NotDefinedAttributeError(Exception):
    pass


class GatewayDefinition(object):

    def __setattr__(self, key, value):
        if key in self.__dict__:
            if isinstance(self.__annotations__[key], value):
                self.__setattr__(key, value)
            else:
                raise PredefinedTypeError
        else:
            raise NotDefinedAttributeError


class Tag(object):

    @property
    def name(self):
        if hasattr(self, "_tag"):
            return self._tag
        else:
            return ""

    @property
    def bytes(self):
        if hasattr(self, "value"):
            return bytes(self.value)
        else:
            return bytes()


class String(str, Tag):

    def __init__(self, tag: str, value=None):
        super(String, self).__init__(value=str(value))
        self._tag = tag

    @property
    def value(self):
        return self.__str__()

    @value.setter
    def value(self, v: str):
        super(String, self).__init__(value=v)


class Slice(list, Tag):

    def __init__(self, tag: str, value: list = None):
        super(Slice, self).__init__()
        if value:
            self.extend(value)
        self._tag = tag

    @property
    def value(self):
        new = list()
        for i in self:
            new.append(i)
        return new

    @value.setter
    def value(self, v: list):
        self.clear()
        self.extend(v)

    @property
    def bytes(self):
        return bytes(json.dumps(self.value))


class Integer(int, Tag):

    def __init__(self, tag: str, value: int = 0):
        super(Integer, self).__init__(x=value)
        self._tag = tag

    @property
    def value(self):
        return self

    @value.setter
    def value(self, v: int):
        super(Integer, self).__init__(x=v)


class Boolean(bool, Tag):

    def __init__(self, tag: str):
        super(Boolean, self).__init__()
        self._tag = tag

    @property
    def bytes(self):
        return bytes(1) if self is True else bytes(0)


class HealthCheckDefinition(GatewayDefinition):
    id = String("ID")
    path = String("Path")
    timeout = Integer("Timeout")
    interval = Integer("Interval")
    retry = Integer("Retry")
    retry_time = Integer("RetryTime")

    def __init__(self, path: String, timeout: Integer,
                 interval: Integer, retry: bool, retry_time: Integer):
        uid = str(uuid.getnode())
        self.id.value = uid
        self.path.value = path
        self.timeout.value = timeout
        self.interval.value = interval
        self.retry.value = Integer("Retry", 1 if retry else 0)
        self.retry_time.value = retry_time


class NodeDefinition(GatewayDefinition):
    id = String("ID")
    name = String("Name")
    host = String("Host")
    port = Integer("Port")
    status = Integer("Status")
    health_check = String("HealthCheck")

    def __init__(self, name: String, host: String, port: Integer, status: Integer, health_check: HealthCheckDefinition):
        uid = str(uuid.getnode())
        self.id.value = uid
        self.name.value = name + uid
        self.host.value = host
        self.port.value = port
        self.status.value = status
        self.health_check.value = health_check.id


class ServiceDefinition(GatewayDefinition):
    name = String("Name")
    node = Slice("Node")

    def __init__(self, name: String):
        self.name.value = name


class RouterDefinition(GatewayDefinition):
    id = String("ID")
    name = String("Name")
    status = String("Status")
    frontend = String("FrontendApi")
    backend = String("BackendApi")
    service = String("Service")

    def __init__(self, name: String, frontend: String, backend: String, service: ServiceDefinition):
        uid = str(uuid.getnode())
        self.id.value = uid
        self.name.value = name
        self.status.value = "0"
        self.frontend.value = frontend
        self.backend.value = backend
        self.service.value = service.name


class EtcdConf(object):
    host: str
    port: int
    user: str
    password: str

    def __str__(self):
        return "HOST: {}, PORT: {}, USER: {}, PASSWORD: {}".format(self.host, str(self.port), self.user, self.password)


class ApiGatewayRegistrant(object):

    def __init__(self, service: ServiceDefinition, node: NodeDefinition,
                 hc: HealthCheckDefinition, etcd_conf: EtcdConf, router: list):
        self._client = None
        self._router = list()
        self._hc = hc
        self._node = node
        self._service = service

        ec = etcd_conf
        try:
            self._client = etcd3.client(host=ec.host, port=ec.port, user=ec.user,
                                        password=ec.password)
        except Exception as e:
            logging.exception(str(e))
            self._client = None
        for r in router:
            if not isinstance(r, RouterDefinition):
                raise PredefinedTypeError
            self._router.append(r)

    def register(self):

        if self._client is None:
            raise ClientNotExist

        self._register_node()
        self._register_service()
        self._register_router()
        self._register_hc()

    def _register_node(self):
        if self._client is None or not isinstance(self._client, etcd3.Etcd3Client):
            raise ClientNotExist

        n = self._node
        c = self._client

        def put(cli, node_id, attr):
            cli.put(ROOT + SLASH.join([NODE_KEY, NODE_PREFIX + node_id, attr.name]), attr.bytes)

        v = c.get(ROOT + SLASH.join([NODE_KEY, NODE_PREFIX + n.id.value, n.id.name]))
        if n.id.value == v:
            # node exists, update node
            pass
        else:
            # node not exists, put new one
            put(c, n.id.value, n.id)
        put(c, n.id.value, n.name)
        put(c, n.id.value, n.host)
        put(c, n.id.value, n.port)
        put(c, n.id.value, n.status)
        put(c, n.id.value, n.health_check)

    def _register_service(self):
        if self._client is None or not isinstance(self._client, etcd3.Etcd3Client):
            raise ClientNotExist

        s = self._service
        c = self._client

        def put(cli, service_name, attr):
            cli.put(ROOT + SLASH.join([SERVICE_KEY, SERVICE_PREFIX + service_name, attr.name]), attr.bytes)

        v = c.get(ROOT + SLASH.join([SERVICE_KEY, SERVICE_PREFIX + s.name, s.name.name]))
        if s.name.value == v:
            # service exists, check node exists in service.node_slice
            node_slice_json = c.get(ROOT + SLASH.join([SERVICE_KEY, SERVICE_PREFIX + s.name.value, s.node.name]))
            try:
                node_slice = json.loads(node_slice_json)
            except Exception as e:
                logging.exception(e)
                node_slice = list()

            flag = 0
            for n in node_slice:
                if n == self._node.id.value:
                    flag = 1
                    break
            if flag == 0 and len(node_slice) > 0:
                node_slice.append(self._node.id.value)
                c.put(ROOT + SLASH.join([SERVICE_KEY, SERVICE_PREFIX + s.name.value, s.node.name]),
                      bytes(json.dumps(node_slice)))
        else:
            # service not exits, put new one
            put(c, s.name.value, s.name)
            s.node.value = [self._node.id.value]
            put(c, s.name.value, s.node)

    def _register_router(self):
        if self._client is None or not isinstance(self._client, etcd3.Etcd3Client):
            raise ClientNotExist

        c = self._client
        rl = self._router

        def put(cli, router_name, attr):
            cli.put(ROOT + SLASH.join([ROUTER_KEY, ROUTER_PREFIX + router_name, attr.name]), attr.bytes)

        for r in rl:
            v = c.get(ROOT + SLASH.join([ROUTER_KEY, ROUTER_PREFIX + r.name.value, r.name.name]))
            if r.name.value == v:
                stat = c.get(ROOT + SLASH.join([ROUTER_KEY, ROUTER_PREFIX + r.name.value, r.status.name]))
                if str(stat) != "1":
                    # router still online, can not be updated
                    continue
            else:
                put(c, r.name.value, r.name)

            put(c, r.name.value, r.id)
            put(c, r.name.value, r.status)
            put(c, r.name.value, r.service)
            put(c, r.name.value, r.frontend)
            put(c, r.name.value, r.backend)

    def _register_hc(self):
        if self._client is None or not isinstance(self._client, etcd3.Etcd3Client):
            raise ClientNotExist

        c = self._client
        hc = self._hc

        def put(cli, hc_id, attr):
            cli.put(ROOT + SLASH.join([HEALTH_CHECK_KEY, HEALTH_CHECK_PREFIX + hc_id, attr.name]), attr.bytes)

        v = c.get(ROOT + SLASH.join([HEALTH_CHECK_KEY, HEALTH_CHECK_PREFIX + hc.id.value, hc.id.name]))
        if hc.id.value == v:
            # health check exists, update
            pass
        else:
            put(c, hc.id.value, hc.id)

        put(c, hc.id.value, hc.path)
        put(c, hc.id.value, hc.interval)
        put(c, hc.id.value, hc.timeout)
        put(c, hc.id.value, hc.retry)
        put(c, hc.id.value, hc.retry_time)