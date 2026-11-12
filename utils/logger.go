//
// Created by HuyN on 2026/10/17.
// Description: Cung cấp các hàm ghi log chuẩn hóa kèm nhãn mức độ (INFO, WARN, ERROR) và dấu thời gian thống nhất
// phục vụ theo dõi vận hành trên server.
//

package utils

import "time"
import "github.com/theritikchoure/logx"

type Logger struct {
	Message string
}

func (logger Logger) Info(message string) {
	logx.ColoringEnabled = true
	logger.Message = message
	logx.Log("[INFO] "+time.Now().String()+" "+message, logx.FGBLUE, logx.BGBLACK)
}

func (logger Logger) Error(message string) {
	logx.ColoringEnabled = true
	logger.Message = message
	logx.Log("[ERROR] "+time.Now().String()+" "+message, logx.FGRED, logx.BGBLACK)
}

func (logger Logger) Warn(message string) {
	logx.ColoringEnabled = true
	logger.Message = message
	logx.Log("[WARN] "+time.Now().String()+" "+message, logx.FGYELLOW, logx.BGBLACK)
}

func (logger Logger) Debug(message string) {
	logx.ColoringEnabled = true
	logger.Message = message
	logx.Log("[DEBUG] "+time.Now().String()+" "+message, logx.FGGREEN, logx.BGBLACK)
}
