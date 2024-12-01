package rtsp

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/yapingcat/gomedia/go-rtsp/sdp"
)

type RtspServer struct {
	buffer      bytes.Buffer
	tracks      map[string]*RtspTrack
	userName    string
	passwd      string
	realm       string
	auth        authenticate
	output      OutPutCallBack
	handle      ServerHandle
	sessionId   string
	sdpContext  *sdp.Sdp
	interleaved int
	isRecord    bool
}

type ServerOption func(*RtspServer)

func WithUserInfo(userName, passwd string) ServerOption {
	return func(rs *RtspServer) {
		rs.userName = userName
		rs.passwd = passwd
	}
}

func WithAuthType(authType string) ServerOption {
	return func(rs *RtspServer) {
		rs.auth = createAuthByAuthenticate(authType)
	}
}

func WithRealm(realm string) ServerOption {
	return func(rs *RtspServer) {
		rs.realm = realm
	}
}

func NewRtspServer(handle ServerHandle, opt ...ServerOption) *RtspServer {
	server := &RtspServer{
		handle:     handle,
		auth:       nil,
		realm:      "gomedia server",
		tracks:     make(map[string]*RtspTrack),
		sdpContext: &sdp.Sdp{},
		isRecord:   false,
	}
	for _, o := range opt {
		o(server)
	}
	if server.auth == nil && server.userName != "" && server.passwd != "" {
		server.auth = createAuthByAuthenticate("Digest")
		server.auth.setUserInfo(server.userName, server.passwd)
		server.auth.setRealm(server.realm)
	}
	server.sdpContext.Attrs = make(map[string]string)
	server.sdpContext.Attrs["control"] = "*"
	return server
}

func (server *RtspServer) AddTrack(track *RtspTrack) {
	track.uri = fmt.Sprintf("track%d", len(server.tracks))
	server.tracks[track.TrackName] = track
	server.sdpContext.ParserSdp(track.mediaDescripe())
}

func (server *RtspServer) SetOutput(output OutPutCallBack) {
	server.output = output
}

func (server *RtspServer) Input(data []byte) (err error) {
	var buf []byte
	// 获取缓冲区数据，在一起处理
	if server.buffer.Len() > 0 {
		server.buffer.Write(data)
		buf = server.buffer.Bytes()
	} else {
		buf = data
	}

	// 有数据
	for len(buf) > 0 {
		// ret 是当前处理的字节
		ret := 0
		if buf[0] == '$' { // RTP 请求，获取 rtp 包
			ret, err = server.hanleRtpOverRtsp(buf)
		} else { // RTSP 消息，比如 play
			ret, err = server.handleRtspMessage(buf)
		}
		if err != nil {
			break
		}
		// 将剩下的消息放入缓冲区，会继续处理
		buf = buf[ret:]
	}

	if err != nil {
		if errors.Is(err, errNeedMore) {
			err = nil
		} else {
			return
		}
	}

	// 缓冲区为 0 则进行重置
	if len(buf) == 0 {
		server.buffer.Reset()
	} else {
		if server.buffer.Len() > 0 {
			server.buffer.Reset()
		}
		server.buffer.Write(buf)
	}
	return nil
}

func (server *RtspServer) hanleRtpOverRtsp(packet []byte) (int, error) {
	// 少于4字节不解析
	if len(packet) < 4 {
		return 0, errNeedMore
	}
	// 第一个字节为 channel 标识
	channel := packet[1]
	// 2、3 字节为长度
	length := binary.BigEndian.Uint16(packet[2:])
	// 当前的包少于对应长度
	if len(packet)-4 < int(length) {
		return 0, errNeedMore
	}

	// 处理 rtp 包，找到对应的 track（轨道） 来处理
	// 常用 track：视频track、音频track、字幕track
	// 视频监控：一个track可能包含视频
	// 视频通话：可能有视频和音频两个track
	// 直播：可能有主视频、音频、备用视频等多个track
	for _, track := range server.tracks {
		isRtcp := false
		// rtcp 包
		if track.transport.Interleaved[1] == int(channel) {
			isRtcp = true
		}

		// 处理
		if track.transport.Interleaved[0] == int(channel) || isRtcp {
			return 4 + int(length), track.Input(packet[4:4+length], isRtcp)
		}
	}
	//improve compatibility
	return 4 + int(length), nil
}

func (server *RtspServer) handleRtspMessage(msg []byte) (int, error) {
	idx := bytes.IndexFunc(msg, func(r rune) bool {
		if r == ' ' {
			return false
		} else {
			return true
		}
	})

	msg = msg[idx:]
	if bytes.HasPrefix(msg, []byte{'R', 'T', 'S', 'P'}) { // 客户端响应
		return server.handleResponse(msg)
	} else { // 客户端请求
		return server.handleRequest(msg)
	}
}

// TODO
// server send request to client
func (server *RtspServer) handleResponse(res []byte) (ret int, err error) {
	response := RtspResponse{}
	ret, err = response.parse(string(res))
	return
}

func (server *RtspServer) handleRequest(req []byte) (ret int, err error) {
	request := RtspRequest{}
	request.Fileds = make(HeadFiled)
	ret, err = request.parse(string(req))
	if err != nil {
		return
	}
	if server.userName != "" && server.passwd != "" {
		server.auth.setMethod(request.Method)
		if !request.Fileds.Has(Authorization) || !server.auth.check(request.Fileds[Authorization]) {
			return ret, server.handleUnAuth(request)
		}
	}

	res := RtspResponse{}
	res.Fileds = make(HeadFiled)
	res.StatusCode = 200
	res.Version = RTSP_1_0
	if server.sessionId != "" {
		if !request.Fileds.Has(Session) || request.Fileds[Session] != server.sessionId {
			res.StatusCode = Session_Not_Found
			return ret, server.sendRespones(request, res)
		}
	}
	switch request.Method {
	case OPTIONS:
		methods := []string{OPTIONS, SET_PARAMETER, GET_PARAMETER, SETUP, DESCRIBE, PLAY, ANNOUNCE, RECORD, TEARDOWN, PAUSE}
		public := strings.Join(methods, ",")
		server.handle.HandleOption(server, request, &res)
		if res.StatusCode == 200 {
			res.Fileds[Public] = public
		}
	case DESCRIBE:
		server.handle.HandleDescribe(server, request, &res)
		if res.StatusCode == OK {
			res.Body = server.sdpContext.Encode()
			res.Fileds[ContentType] = "application/sdp"
		}
	case SETUP:
		foundTrack := false
		for _, track := range server.tracks {
			if !strings.Contains(request.Uri, track.uri) {
				continue
			}
			foundTrack = true
			track.uri = request.Uri
			transport := NewRtspTransport()
			transport.DecodeString(request.Fileds[Transport])
			server.handle.HandleSetup(server, request, &res, transport, track)
			if res.StatusCode == 200 {
				if server.sessionId == "" {
					number := []byte("0123456789")
					b := make([]byte, 10)
					for i := range b {
						b[i] = number[rand.Intn(len(number))]
					}
					server.sessionId = string(b)
				}
				if transport.Proto == TCP {
					transport.Interleaved[0] = server.interleaved
					transport.Interleaved[1] = server.interleaved + 1
					server.interleaved = server.interleaved + 2
					track.OnPacket(func(b []byte, isRtcp bool) error {
						interleavedPacket := make([]byte, 4+len(b))
						interleavedPacket[0] = '$'
						if isRtcp {
							interleavedPacket[1] = byte(transport.Interleaved[1])
						} else {
							interleavedPacket[1] = byte(transport.Interleaved[0])
						}
						binary.BigEndian.PutUint16(interleavedPacket[2:], uint16(len(b)))
						copy(interleavedPacket[4:], b)
						return server.output(interleavedPacket)
					})
				}
				res.Fileds[Transport] = transport.EncodeString()
				res.Fileds[Session] = server.sessionId
				track.SetTransport(transport)
			}
			break
		}
		if !foundTrack {
			res.StatusCode = BAD_REQUEST
		}
	case ANNOUNCE:
		if err = server.sdpContext.ParserSdp(request.Body); err != nil {
			return
		}
		server.isRecord = true
		for _, media := range server.sdpContext.Medias {
			fmtpHandle := sdp.CreateFmtpParamParser(media.EncodeName)
			if fmtpHandle != nil {
				fmtpHandle.Load(media.Attrs["fmtp"])
			}
			var track *RtspTrack = nil
			if media.MediaType == "audio" {
				track = NewAudioTrack(NewAudioCodec(media.EncodeName, uint8(media.PayloadType), uint32(media.ClockRate), media.ChannelCount), WithCodecParamHandler(fmtpHandle))
			} else if media.MediaType == "video" {
				track = NewVideoTrack(NewVideoCodec(media.EncodeName, uint8(media.PayloadType), uint32(media.ClockRate)), WithCodecParamHandler(fmtpHandle))
			} else {
				track = NewMetaTrack(NewApplicatioCodec(media.EncodeName, uint8(media.PayloadType)))
			}
			track.uri = media.ControlUrl
			server.tracks[media.MediaType] = track
		}
		server.handle.HandleAnnounce(server, request, server.tracks)
	case PLAY:
		var tr *RangeTime = nil
		var info []*RtpInfo
		if request.Fileds.Has(Range) {
			tr, _ = parseRange(request.Fileds[Range])
		}
		for _, t := range server.tracks {
			i := &RtpInfo{}
			i.Url = t.uri
			i.Seq = t.initSequence
		}
		server.handle.HandlePlay(server, request, &res, tr, info)
		if res.StatusCode == 200 {
			if tr != nil {
				res.Fileds[Range] = tr.EncodeString()
			}
			if len(info) > 0 {
				infostr := ""
				for _, i := range info {
					infostr += i.EncodeString()
					infostr += ","
				}
				res.Fileds[RTPInfo] = infostr[:len(infostr)-1]
			}
		}
	case RECORD:
		var tr *RangeTime = nil
		var info []*RtpInfo
		if request.Fileds.Has(Range) {
			tr, _ = parseRange(request.Fileds[Range])
		}
		for _, t := range server.tracks {
			i := &RtpInfo{}
			i.Url = t.uri
			i.Seq = t.initSequence
		}
		server.handle.HandleRecord(server, request, &res, tr, info)
		if res.StatusCode == 200 {
			if tr != nil {
				res.Fileds[Range] = tr.EncodeString()
			}
			if len(info) > 0 {
				infostr := ""
				for _, i := range info {
					infostr += i.EncodeString()
					infostr += ","
				}
				res.Fileds[RTPInfo] = infostr[:len(infostr)-1]
			}
		}
	case TEARDOWN:
		server.handle.HandleTeardown(server, request, &res)
	case PAUSE:
		server.handle.HandlePause(server, request, &res)
	case SET_PARAMETER:
		server.handle.HandleSetParameter(server, request, &res)
	case GET_PARAMETER:
		server.handle.HandleGetParameter(server, request, &res)
	}
	return ret, server.sendRespones(request, res)
}

func (server *RtspServer) handleUnAuth(request RtspRequest) error {
	response := RtspResponse{Fileds: make(HeadFiled)}
	response.StatusCode = 401
	response.Fileds[WWWAuthenticate] = server.auth.wwwAuthenticate()
	return server.sendRespones(request, response)
}

func (server *RtspServer) sendRespones(req RtspRequest, res RtspResponse) error {
	res.Fileds[CSeq] = req.Fileds[CSeq]
	res.Fileds[Date] = time.Now().UTC().Format("02 Jan 06 15:04:05 GMT")
	if server.output != nil {
		return server.output([]byte(res.Encode()))
	}
	return nil
}
