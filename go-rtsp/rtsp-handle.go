package rtsp

import "github.com/yapingcat/gomedia/go-rtsp/sdp"

// ClientHandle 定义了 RTSP 客户端处理各种 RTSP 请求和响应的接口方法
type ClientHandle interface {
	// HandleOption 处理服务器的 OPTIONS 方法响应，通常用于查询服务器支持的方法
	// cli: RTSP 客户端实例
	// res: 服务器返回的响应
	// public: 服务器支持的公共方法列表
	HandleOption(cli *RtspClient, res RtspResponse, public []string) error

	// HandleDescribe 处理 DESCRIBE 方法的响应，用于获取媒体流描述信息
	// cli: RTSP 客户端实例
	// res: 服务器返回的响应
	// sdp: 会话描述协议(SDP)信息
	// tracks: 媒体轨道映射
	HandleDescribe(cli *RtspClient, res RtspResponse, sdp *sdp.Sdp, tracks map[string]*RtspTrack) error

	// HandleSetup 处理 SETUP 方法响应，配置传输参数和会话
	// cli: RTSP 客户端实例
	// res: 服务器返回的响应
	// currentTrack: 当前正在设置的媒体轨道
	// tracks: 所有媒体轨道映射
	// sessionId: 会话标识符
	// timeout: 会话超时时间
	HandleSetup(cli *RtspClient, res RtspResponse, currentTrack *RtspTrack, tracks map[string]*RtspTrack, sessionId string, timeout int) error

	// HandleAnnounce 处理 ANNOUNCE 方法响应，通常用于推流前的媒体信息通告
	// cli: RTSP 客户端实例
	// res: 服务器返回的响应
	HandleAnnounce(cli *RtspClient, res RtspResponse) error

	// HandlePlay 处理 PLAY 方法响应，开始播放媒体流
	// cli: RTSP 客户端实例
	// res: 服务器返回的响应
	// timeRange: 播放时间范围
	// info: RTP 播放信息
	HandlePlay(cli *RtspClient, res RtspResponse, timeRange *RangeTime, info *RtpInfo) error

	// HandlePause 处理 PAUSE 方法响应，暂停媒体流
	// cli: RTSP 客户端实例
	// res: 服务器返回的响应
	HandlePause(cli *RtspClient, res RtspResponse) error

	// HandleTeardown 处理 TEARDOWN 方法响应，结束媒体会话
	// cli: RTSP 客户端实例
	// res: 服务器返回的响应
	HandleTeardown(cli *RtspClient, res RtspResponse) error

	// HandleGetParameter 处理 GET_PARAMETER 方法响应，获取特定参数
	// cli: RTSP 客户端实例
	// res: 服务器返回的响应
	HandleGetParameter(cli *RtspClient, res RtspResponse) error

	// HandleSetParameter 处理 SET_PARAMETER 方法响应，设置特定参数
	// cli: RTSP 客户端实例
	// res: 服务器返回的响应
	HandleSetParameter(cli *RtspClient, res RtspResponse) error

	// HandleRedirect 处理重定向响应
	// cli: RTSP 客户端实例
	// req: 原始请求
	// location: 重定向地址
	// timeRange: 重定向时间范围
	HandleRedirect(cli *RtspClient, req RtspRequest, location string, timeRange *RangeTime) error

	// HandleRecord 处理 RECORD 方法响应，开始录制媒体流
	// cli: RTSP 客户端实例
	// res: 服务器返回的响应
	// timeRange: 录制时间范围
	// info: RTP 录制信息
	HandleRecord(cli *RtspClient, res RtspResponse, timeRange *RangeTime, info *RtpInfo) error

	// HandleRequest 处理通用请求
	// cli: RTSP 客户端实例
	// req: 原始请求
	HandleRequest(cli *RtspClient, req RtspRequest) error
}

// ServerHandle 定义了 RTSP 服务端处理各种 RTSP 请求和响应的接口方法
type ServerHandle interface {
	// HandleOption 处理客户端的 OPTIONS 方法请求
	// svr: RTSP 服务器实例
	// req: 客户端请求
	// res: 服务器响应
	HandleOption(svr *RtspServer, req RtspRequest, res *RtspResponse)

	// HandleDescribe 处理客户端的 DESCRIBE 方法请求，返回媒体流描述
	// svr: RTSP 服务器实例
	// req: 客户端请求
	// res: 服务器响应
	HandleDescribe(svr *RtspServer, req RtspRequest, res *RtspResponse)

	// HandleSetup 处理客户端的 SETUP 方法请求，配置传输参数
	// svr: RTSP 服务器实例
	// req: 客户端请求
	// res: 服务器响应
	// transport: RTSP 传输参数
	// tracks: 媒体轨道
	HandleSetup(svr *RtspServer, req RtspRequest, res *RtspResponse, transport *RtspTransport, tracks *RtspTrack)

	// HandleAnnounce 处理客户端的 ANNOUNCE 方法请求，接收媒体流信息
	// svr: RTSP 服务器实例
	// req: 客户端请求
	// tracks: 媒体轨道映射
	HandleAnnounce(svr *RtspServer, req RtspRequest, tracks map[string]*RtspTrack)

	// HandlePlay 处理客户端的 PLAY 方法请求，开始播放媒体流
	// svr: RTSP 服务器实例
	// req: 客户端请求
	// res: 服务器响应
	// timeRange: 播放时间范围
	// info: RTP 播放信息列表
	HandlePlay(svr *RtspServer, req RtspRequest, res *RtspResponse, timeRange *RangeTime, info []*RtpInfo)

	// HandlePause 处理客户端的 PAUSE 方法请求，暂停媒体流
	// svr: RTSP 服务器实例
	// req: 客户端请求
	// res: 服务器响应
	HandlePause(svr *RtspServer, req RtspRequest, res *RtspResponse)

	// HandleTeardown 处理客户端的 TEARDOWN 方法请求，结束媒体会话
	// svr: RTSP 服务器实例
	// req: 客户端请求
	// res: 服务器响应
	HandleTeardown(svr *RtspServer, req RtspRequest, res *RtspResponse)

	// HandleGetParameter 处理客户端的 GET_PARAMETER 方法请求，获取参数
	// svr: RTSP 服务器实例
	// req: 客户端请求
	// res: 服务器响应
	HandleGetParameter(svr *RtspServer, req RtspRequest, res *RtspResponse)

	// HandleSetParameter 处理客户端的 SET_PARAMETER 方法请求，设置参数
	// svr: RTSP 服务器实例
	// req: 客户端请求
	// res: 服务器响应
	HandleSetParameter(svr *RtspServer, req RtspRequest, res *RtspResponse)

	// HandleRecord 处理客户端的 RECORD 方法请求，开始录制媒体流
	// svr: RTSP 服务器实例
	// req: 客户端请求
	// res: 服务器响应
	// timeRange: 录制时间范围
	// info: RTP 录制信息列表
	HandleRecord(svr *RtspServer, req RtspRequest, res *RtspResponse, timeRange *RangeTime, info []*RtpInfo)

	// HandleResponse 处理服务器的响应
	// svr: RTSP 服务器实例
	// res: 服务器响应
	HandleResponse(svr *RtspServer, res RtspResponse)
}
