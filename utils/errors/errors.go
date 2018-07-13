package errors

import (
	"fmt"
	"log"
)

const (
	errorStringFormat = "ErrCode: {%d}, ErrMsg: {%s}, ErrMsgEn: {%s}"
)

// Error holds the error message, this message never really changes
type Error struct {
	ErrCode int `json:"err_code"`
	// The message of the error.
	ErrMsg string `json:"err_msg"`
	// The message of the error.
	ErrMsgEn string `json:"err_msg_en"`
}

type CustomErrMsg BaseErrMsg

// New creates and returns an Error with a pre-defined user output message
func New(errCode int, errMsg ...CustomErrMsg) Error {
	if _, ok := BaseErrors[errCode]; ok {
		if len(errMsg) > 1 {
			log.SetPrefix("[WARNING]")
			log.Print("redundant errMsg parameters")
		}

		if len(errMsg) > 0 {
			var tmpErrMsg string
			var tmpErrMsgEn string

			if errMsg[0].ErrMsg != "" {
				tmpErrMsg = errMsg[0].ErrMsg
			} else {
				tmpErrMsg = BaseErrors[errCode].ErrMsg
			}
			if errMsg[0].ErrMsgEn != "" {
				tmpErrMsgEn = errMsg[0].ErrMsgEn
			} else {
				tmpErrMsgEn = BaseErrors[errCode].ErrMsgEn
			}
			return Error{
				ErrCode:  errCode,
				ErrMsg:   tmpErrMsg,
				ErrMsgEn: tmpErrMsgEn,
			}
		} else {
			return Error{
				ErrCode:  errCode,
				ErrMsg:   BaseErrors[errCode].ErrMsg,
				ErrMsgEn: BaseErrors[errCode].ErrMsgEn,
			}
		}
	} else {
		if len(errMsg) == 0 {
			log.SetPrefix("[WARNING]")
			log.Print("lack of err msg info when defining cumstom error")
			return Error{
				ErrCode:  errCode,
				ErrMsg:   "未知错误",
				ErrMsgEn: "unknown error",
			}
		}
		if errMsg[0].ErrMsg == "" {
			log.SetPrefix("[WARNING]")
			log.Print("lack of err msg info when defining cumstom error")
			return Error{
				ErrCode:  errCode,
				ErrMsg:   "未知错误",
				ErrMsgEn: "unknown error",
			}
		}
		return Error{
			ErrCode:  errCode,
			ErrMsg:   errMsg[0].ErrMsg,
			ErrMsgEn: errMsg[0].ErrMsgEn,
		}
	}
}

func (e Error) Error() string {
	return fmt.Sprintf(errorStringFormat, e.ErrCode, e.ErrMsg, e.ErrMsgEn)
}
