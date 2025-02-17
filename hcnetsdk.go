package hcnetsdk

/*
#cgo CFLAGS: -I./include
#cgo windows LDFLAGS: -L${SRCDIR}/libs/windows -lHCCore -lHCNetSDK
#cgo linux LDFLAGS: -L${SRCDIR}/libs/linux -lHCCore -lHCNetSDK
#cgo darwin LDFLAGS: -L${SRCDIR}/libs/darwin -lHCCore -lHCNetSDK
#include "HCNetSDK.h"
// 消息回调函数
extern BOOL fMSGCallBackGo(LONG lCommand, NET_DVR_ALARMER *pAlarmer, char *pAlarmInfo, DWORD dwBufLen, void* pUser);
BOOL MSGCallBackCgo(LONG lCommand, NET_DVR_ALARMER *pAlarmer, char *pAlarmInfo, DWORD dwBufLen, void* pUser)
{
    return fMSGCallBackGo(lCommand, pAlarmer, pAlarmInfo, dwBufLen, pUser);
}
// 异常回调函数
extern void fExceptionCallBackGo(DWORD dwType, LONG lUserID, LONG lHandle, void *pUser);
void FExceptionCallBackCgo(DWORD dwType, LONG lUserID, LONG lHandle, void *pUser)
{
    fExceptionCallBackGo(dwType, lUserID, lHandle, pUser);
}
// 视频实时播放
extern void fRealDataCallBackGo(LONG lPlayHandle, DWORD dwDataType, BYTE *pBuffer, DWORD dwBufSize, void* pUser);
void FRealDataCallBackCgo(LONG lPlayHandle, DWORD dwDataType, BYTE *pBuffer, DWORD dwBufSize, void* pUser)
{
    fRealDataCallBackGo(lPlayHandle, dwDataType, pBuffer, dwBufSize, pUser);
}
*/
import "C"
import (
	"errors"
	"fmt"
	"time"
	"unsafe"
)

func init() {
	Init()
}

// 初始化
func Init() {
	C.NET_DVR_Init()
}

// 清除释放资源
func Close() {
	C.NET_DVR_Cleanup()
}

// 登录设备
func Login(ip string, port int, username string, password string) (int, error) {
	cIp := C.CString(ip)
	defer C.free(unsafe.Pointer(cIp))
	cUsername := C.CString(username)
	defer C.free(unsafe.Pointer(cUsername))
	cPassword := C.CString(password)
	defer C.free(unsafe.Pointer(cPassword))
	var deviceInfo C.NET_DVR_DEVICEINFO_V30

	cUserId := C.NET_DVR_Login_V30(cIp, C.WORD(port), cUsername, cPassword, &deviceInfo)
	if cUserId < 0 {
		return -1, LastError()
	}
	return int(cUserId), nil
}

// 注销登录
func Logout(userId int) error {
	if userId > -1 {
		cResult := C.NET_DVR_Logout(C.LONG(userId))
		if cResult != 1 {
			return LastError()
		}
	}
	return nil
}

// 获取错误信息
func LastError() error {
	cCode := C.NET_DVR_GetLastError()
	if cCode == 0 {
		return nil
	}

	var text string
	if cCode == 3 {
		text = "sdk not init."
	} else {
		cCode2 := C.LONG(cCode)
		cText := C.NET_DVR_GetErrorMsg(&cCode2)
		text = C.GoString(cText)
	}
	return NewHcnetError(int(cCode), text)
}

// 执行操作
func DoAction(ip string, port int, username string, password string, action func(int) (interface{}, error)) (interface{}, error) {
	userId, err := Login(ip, port, username, password)
	if err != nil {
		return nil, err
	}
	defer Logout(userId)
	return action(userId)
}

// // 设备布防
// func SetupAlarm(userId int, messageCallback MessageCallBack, exceptionCallback ExceptionCallBack, data unsafe.Pointer) (int, error) {
// 	messageCallBackHooks[userId] = messageCallback
// 	cUserId := C.LONG(userId)
// 	cResult := C.NET_DVR_SetDVRMessageCallBack_V30(C.MSGCallBack(C.MSGCallBackCgo), data)
// 	if cResult != 1 {
// 		return -1, LastError()
// 	}
// 	cHandle := C.NET_DVR_SetupAlarmChan_V30(cUserId)
// 	if cHandle == -1 {
// 		return -1, LastError()
// 	}

// 	// 异常回调
// 	exceptionCallBackHooks[userId] = exceptionCallback
// 	if exceptionCallback != nil {
// 		cExceptionResult := C.NET_DVR_SetExceptionCallBack_V30(C.UINT(0), nil, C.FExceptionCallBack(C.FExceptionCallBackCgo), nil)
// 		if cExceptionResult == 0 {
// 			C.NET_DVR_CloseAlarmChan_V30(cHandle)
// 			return int(cHandle), LastError()
// 		}
// 	}
// 	return int(cHandle), nil
// }

// 关闭报警布防
// func CloseAlarm(handle int) error {
// 	cResult := C.NET_DVR_CloseAlarmChan_V30(C.LONG(handle))
// 	if cResult != 1 {
// 		return LastError()
// 	}
// 	return nil
// }

// 视频实时播放
func RealPlay(userId int, callback RealDataCallBack, data interface{}) (int, error) {
	realDataCallBackHooks[userId] = callback

	previewInfo := &C.NET_DVR_PREVIEWINFO{}
	previewInfo.lChannel = C.LONG(1)
	previewInfo.dwStreamType = C.DWORD(0)
	previewInfo.dwLinkMode = C.DWORD(1)
	previewInfo.hPlayWnd = nil
	previewInfo.bBlocked = C.DWORD(0)
	previewInfo.bPassbackRecord = C.DWORD(0)
	previewInfo.byPreviewMode = C.BYTE(0)

	// 自定义数据
	userData := realDataCallBackDataWrapper{userId, data}
	cHandle := C.NET_DVR_RealPlay_V40(C.LONG(userId), previewInfo, C.REALDATACALLBACK(C.FRealDataCallBackCgo), unsafe.Pointer(&userData))
	if cHandle == -1 {
		return 0, LastError()
	}
	return int(cHandle), nil
}

// 停止视频流播放
func StopRealPlay(handle int) (bool, error) {
	cResult := C.NET_DVR_StopRealPlay(C.LONG(handle))
	if cResult != 1 {
		return false, LastError()
	}
	return true, nil
}

// 获取设备DvrConfig
func GetDvrConfig(userId int, channel int32, command int64, buffer unsafe.Pointer, bufferSize int) error {
	lUserID := C.LONG(userId)
	dwCommand := C.DWORD(command)
	lChannel := C.LONG(channel)
	lpOutBuffer := C.LPVOID(buffer)
	dwOutBufferSize := C.DWORD(bufferSize)
	bytesReturned := C.DWORD(0)
	cResult := C.NET_DVR_GetDVRConfig(lUserID, dwCommand, lChannel, lpOutBuffer, dwOutBufferSize, &bytesReturned)
	if cResult != 1 {
		return LastError()
	}
	return nil
}

// 设置设备DvrConfig
func SetDvrConfig(userId int, channel int32, command int64, buffer unsafe.Pointer) error {
	lUserID := C.LONG(userId)
	dwCommand := C.DWORD(command)
	lChannel := C.LONG(channel)
	lpInBuffer := C.LPVOID(buffer)
	dwInBufferSize := C.DWORD(unsafe.Sizeof(buffer))
	cResult := C.NET_DVR_SetDVRConfig(lUserID, dwCommand, lChannel, lpInBuffer, dwInBufferSize)
	if int(cResult) != 1 {
		return LastError()
	}
	return nil
}

// ISAPI协议透传
func PassThrough(userId int, url string, data string) (*PassThroughResponse, error) {
	// 输入
	var configInput = C.NET_DVR_XML_CONFIG_INPUT{}
	urlBytes := []byte(url)
	cUrlBytes := C.CBytes(urlBytes)
	defer C.free(cUrlBytes)

	dataBytes := []byte(data)
	cDataBytes := C.CBytes(dataBytes)
	defer C.free(cDataBytes)

	configInput.lpRequestUrl = cUrlBytes
	configInput.dwRequestUrlLen = C.DWORD(len(urlBytes))
	configInput.lpInBuffer = cDataBytes
	configInput.dwInBufferSize = C.DWORD(len(dataBytes))

	// 输出
	var configOutput = C.NET_DVR_XML_CONFIG_OUTPUT{}
	outBufferBytes := make([]byte, 3*1024*1024)
	outStatusBufferBytes := make([]byte, 3*1024*1024)
	configOutput.lpOutBuffer = C.CBytes(outBufferBytes)
	configOutput.dwOutBufferSize = C.DWORD(len(outBufferBytes))
	configOutput.lpStatusBuffer = C.CBytes(outStatusBufferBytes)
	configOutput.dwStatusSize = C.DWORD(len(outStatusBufferBytes))

	cResult := C.NET_DVR_STDXMLConfig(C.LONG(userId), &configInput, &configOutput)
	if cResult != 1 {
		return nil, LastError()
	}
	return nil, nil
}

// 设备是否在线
func IsOnline(userId int) bool {
	cResult := C.NET_DVR_RemoteControl(C.LONG(userId), C.DWORD(20005), nil, C.DWORD(0))
	return 1 == cResult
}

// 重启设备
func Reboot(userId int) error {
	cResult := C.NET_DVR_RebootDVR(C.LONG(userId))
	if cResult != 1 {
		return LastError()
	}
	return nil
}

// 设置设备时间
func GetDeviceTime(userId int) (*time.Time, error) {
	netDvrTime := C.NET_DVR_TIME{}
	err := GetDvrConfig(userId, 0, 118, unsafe.Pointer(&netDvrTime), int(unsafe.Sizeof(netDvrTime)))
	if err != nil {
		return nil, err
	}
	date := time.Date(
		int(netDvrTime.dwYear),
		time.Month(netDvrTime.dwMonth),
		int(netDvrTime.dwDay),
		int(netDvrTime.dwHour),
		int(netDvrTime.dwMinute),
		int(netDvrTime.dwSecond),
		0,
		time.UTC,
	)
	return &date, nil
}

// 设置设备时间
func SetDeviceTime(userId int, date time.Time) error {
	netDvrTime := C.NET_DVR_TIME{}
	netDvrTime.dwYear = C.DWORD(date.Year())
	netDvrTime.dwMonth = C.DWORD(date.Month())
	netDvrTime.dwDay = C.DWORD(date.Day())
	netDvrTime.dwHour = C.DWORD(date.Hour())
	netDvrTime.dwMinute = C.DWORD(date.Minute())
	netDvrTime.dwSecond = C.DWORD(date.Second())
	err := SetDvrConfig(userId, 0, 119, unsafe.Pointer(&netDvrTime))
	if err != nil {
		return err
	}
	return nil
}

// 修改密码
func ModifyPassword(userId int, username string, newPassword string) error {
	// 获取原始配置
	cDvrUsers := C.NET_DVR_USER_V30{}
	err := GetDvrConfig(userId, 0, 1006, unsafe.Pointer(&cDvrUsers), int(unsafe.Sizeof(cDvrUsers)))
	if err != nil {
		return err
	}

	for i := 0; i < len(cDvrUsers.struUser); i++ {
		// 账号名处理
		userInfo := &cDvrUsers.struUser[i]
		name := cgoConvertCBytesToString(userInfo.sUserName[:])
		if name == username {
			passwordBytes := []byte(newPassword)
			cPasswordBytes := [16]C.BYTE{}
			for i := 0; i < len(cPasswordBytes) && i < len(passwordBytes); i++ {
				cPasswordBytes[i] = C.BYTE(passwordBytes[i])
			}
			userInfo.sPassword = cPasswordBytes
		}
	}
	return SetDvrConfig(userId, 0, 1007, unsafe.Pointer(&cDvrUsers))
}

// 抓拍接口
// 返回图片内容
func GetJPG(userId int) (string, error) {
	picPath := time.Now().Format("20060102150405") + ".jpeg"
	path := C.CString(picPath)
	defer C.free(unsafe.Pointer(path))
	content := &C.NET_DVR_JPEGPARA{}
	content.wPicSize = C.WORD(10)
	content.wPicQuality = C.WORD(0)
	ok := C.NET_DVR_CaptureJPEGPicture(C.LONG(userId), C.LONG(1), content, path)
	if ok == 1 {
		return picPath, nil
	}
	return "", errors.New("no get")
}

//云台

//--------------------------------------------------------------------
// sdk操作
//--------------------------------------------------------------------

// 获取sdk版本号
func GetSdkVersion() string {
	buildVersion := uint32(C.NET_DVR_GetSDKBuildVersion())
	version := fmt.Sprintf("%d.%d.%d", buildVersion>>24, buildVersion<<8>>24, buildVersion<<16>>16)
	return version
}
