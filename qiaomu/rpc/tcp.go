package rpc

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/binary"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"reflect"
	"sync/atomic"
	"time"

	"github.com/qingbo1011/qiaomu/register"
	"golang.org/x/time/rate"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

//TCP 客户端 服务端
//客户端 1. 连接服务端 2. 发送请求数据 （编码） 二进制 通过网络发送 3. 等待回复 接收到响应（解码）
//服务端 1. 启动服务 2. 接收请求 （解码），根据请求 调用对应的服务 得到响应数据 3. 将响应数据发送给客户端（编码）

type Serializer interface {
	Serialize(data any) ([]byte, error)
	DeSerialize(data []byte, target any) error
}

// GobSerializer Gob协议
type GobSerializer struct{}

func (c GobSerializer) Serialize(data any) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	if err := encoder.Encode(data); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (c GobSerializer) DeSerialize(data []byte, target any) error {
	buffer := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buffer)
	return decoder.Decode(target)
}

type ProtobufSerializer struct{}

func (c ProtobufSerializer) Serialize(data any) ([]byte, error) {
	marshal, err := proto.Marshal(data.(proto.Message))
	if err != nil {
		return nil, err
	}
	return marshal, nil
}

func (c ProtobufSerializer) DeSerialize(data []byte, target any) error {
	message := target.(proto.Message)
	return proto.Unmarshal(data, message)
}

type SerializerType byte

const (
	Gob SerializerType = iota
	ProtoBuff
)

type CompressInterface interface {
	Compress([]byte) ([]byte, error)
	UnCompress([]byte) ([]byte, error)
}

type CompressType byte

const (
	Gzip CompressType = iota
)

type GzipCompress struct{}

func (c GzipCompress) Compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)

	_, err := w.Write(data)
	if err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (c GzipCompress) UnCompress(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	defer reader.Close()
	if err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)
	// 从 Reader 中读取出数据
	if _, err := buf.ReadFrom(reader); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

const MagicNumber byte = 0x1d
const Version = 0x01

type MessageType byte

const (
	msgRequest MessageType = iota
	msgResponse
	msgPing
	msgPong
)

type Header struct {
	MagicNumber   byte
	Version       byte
	FullLength    int32
	MessageType   MessageType
	CompressType  CompressType
	SerializeType SerializerType
	RequestId     int64
}

type QueenRpcMessage struct {
	Header *Header // 消息头
	Data   any     // 消息体
}

type QueenRpcRequest struct {
	RequestId   int64
	ServiceName string
	MethodName  string
	Args        []any
}

type QueenRpcResponse struct {
	RequestId     int64
	Code          int16
	Msg           string
	CompressType  CompressType
	SerializeType SerializerType
	Data          any
}

type QueenRpcServer interface {
	Register(name string, service interface{})
	Run()
	Stop()
}

type QueenTcpServer struct {
	host           string
	port           int
	listen         net.Listener
	serviceMap     map[string]any
	RegisterType   string
	RegisterOption register.Option
	RegisterCli    register.QueenRegister
	LimiterTimeOut time.Duration
	Limiter        *rate.Limiter
}

func NewTcpServer(host string, port int) (*QueenTcpServer, error) {
	listen, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return nil, err
	}
	m := &QueenTcpServer{serviceMap: make(map[string]any)}
	m.listen = listen
	m.port = port
	m.host = host
	return m, nil
}

func (s *QueenTcpServer) SetLimiter(limit, cap int) {
	s.Limiter = rate.NewLimiter(rate.Limit(limit), cap)
}

func (s *QueenTcpServer) Register(name string, service interface{}) {
	t := reflect.TypeOf(service)
	if t.Kind() != reflect.Pointer {
		panic("service must be pointer")
	}
	s.serviceMap[name] = service

	err := s.RegisterCli.CreateCli(s.RegisterOption)
	if err != nil {
		panic(err)
	}
	err = s.RegisterCli.RegisterService(name, s.host, s.port)
	if err != nil {
		panic(err)
	}
}

type QueenTcpConn struct {
	conn    net.Conn
	rspChan chan *QueenRpcResponse
}

func (c QueenTcpConn) Send(rsp *QueenRpcResponse) error {
	if rsp.Code != 200 {
		//进行默认的数据发送
	}
	headers := make([]byte, 17) // 发送编码
	headers[0] = MagicNumber    // magic number
	headers[1] = Version        // version
	//full length
	headers[6] = byte(msgResponse)                                 // 消息类型
	headers[7] = byte(rsp.CompressType)                            // 压缩类型
	headers[8] = byte(rsp.SerializeType)                           // 序列化
	binary.BigEndian.PutUint64(headers[9:], uint64(rsp.RequestId)) // 请求id
	se := loadSerializer(rsp.SerializeType)                        // 编码 先序列化 在压缩
	var body []byte
	var err error
	if rsp.SerializeType == ProtoBuff {
		pRsp := &Response{}
		pRsp.SerializeType = int32(rsp.SerializeType)
		pRsp.CompressType = int32(rsp.CompressType)
		pRsp.Code = int32(rsp.Code)
		pRsp.Msg = rsp.Msg
		pRsp.RequestId = rsp.RequestId
		// value, err := structpb.
		//	log.Println(err)
		m := make(map[string]any)
		marshal, _ := json.Marshal(rsp.Data)
		_ = json.Unmarshal(marshal, &m)
		value, err := structpb.NewStruct(m)
		log.Println(err)
		pRsp.Data = structpb.NewStructValue(value)
		body, err = se.Serialize(pRsp)
	} else {
		body, err = se.Serialize(rsp)
	}
	if err != nil {
		return err
	}
	com := loadCompress(rsp.CompressType)
	body, err = com.Compress(body)
	if err != nil {
		return err
	}
	fullLen := 17 + len(body)
	binary.BigEndian.PutUint32(headers[2:6], uint32(fullLen))

	_, err = c.conn.Write(headers[:])
	if err != nil {
		return err
	}
	_, err = c.conn.Write(body[:])
	if err != nil {
		return err
	}
	return nil
}

func (s *QueenTcpServer) Stop() {
	err := s.listen.Close()
	if err != nil {
		log.Println(err)
	}
}

func (s *QueenTcpServer) Run() {
	for {
		conn, err := s.listen.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		msConn := &QueenTcpConn{conn: conn, rspChan: make(chan *QueenRpcResponse, 1)}
		//1. 一直接收数据 解码工作 请求业务获取结果 发送到rspChan
		//2. 获得结果 编码 发送数据
		go s.readHandle(msConn)
		go s.writeHandle(msConn)
	}
}

func (s *QueenTcpServer) readHandle(conn *QueenTcpConn) {
	defer func() {
		if err := recover(); err != nil {
			log.Println("readHandle recover ", err)
			conn.conn.Close()
		}
	}()
	// 限流处理
	ctx, cancel := context.WithTimeout(context.Background(), s.LimiterTimeOut)
	defer cancel()
	err2 := s.Limiter.WaitN(ctx, 1)
	if err2 != nil {
		rsp := &QueenRpcResponse{}
		rsp.Code = 700 //被限流的错误
		rsp.Msg = err2.Error()
		conn.rspChan <- rsp
		return
	}
	// 接收数据并解码
	msg, err := decodeFrame(conn.conn)
	if err != nil {
		rsp := &QueenRpcResponse{}
		rsp.Code = 500
		rsp.Msg = err.Error()
		conn.rspChan <- rsp
		return
	}
	if msg.Header.MessageType == msgRequest {
		if msg.Header.SerializeType == ProtoBuff {
			req := msg.Data.(*Request)
			rsp := &QueenRpcResponse{RequestId: req.RequestId}
			rsp.SerializeType = msg.Header.SerializeType
			rsp.CompressType = msg.Header.CompressType
			serviceName := req.ServiceName
			service, ok := s.serviceMap[serviceName]
			if !ok {
				rsp := &QueenRpcResponse{}
				rsp.Code = 500
				rsp.Msg = errors.New("no service found").Error()
				conn.rspChan <- rsp
				return
			}
			methodName := req.MethodName
			method := reflect.ValueOf(service).MethodByName(methodName)
			if method.IsNil() {
				rsp := &QueenRpcResponse{}
				rsp.Code = 500
				rsp.Msg = errors.New("no service method found").Error()
				conn.rspChan <- rsp
				return
			}
			//调用方法
			args := make([]reflect.Value, len(req.Args))
			for i := range req.Args {
				of := reflect.ValueOf(req.Args[i].AsInterface())
				of = of.Convert(method.Type().In(i))
				args[i] = of
			}
			result := method.Call(args)

			results := make([]any, len(result))
			for i, v := range result {
				results[i] = v.Interface()
			}
			err, ok := results[len(result)-1].(error)
			if ok {
				rsp.Code = 500
				rsp.Msg = err.Error()
				conn.rspChan <- rsp
				return
			}
			rsp.Code = 200
			rsp.Data = results[0]
			conn.rspChan <- rsp
		} else {
			req := msg.Data.(*QueenRpcRequest)
			rsp := &QueenRpcResponse{RequestId: req.RequestId}
			rsp.SerializeType = msg.Header.SerializeType
			rsp.CompressType = msg.Header.CompressType
			serviceName := req.ServiceName
			service, ok := s.serviceMap[serviceName]
			if !ok {
				rsp := &QueenRpcResponse{}
				rsp.Code = 500
				rsp.Msg = errors.New("no service found").Error()
				conn.rspChan <- rsp
				return
			}
			methodName := req.MethodName
			method := reflect.ValueOf(service).MethodByName(methodName)
			if method.IsNil() {
				rsp := &QueenRpcResponse{}
				rsp.Code = 500
				rsp.Msg = errors.New("no service method found").Error()
				conn.rspChan <- rsp
				return
			}
			//调用方法
			args := req.Args
			var valuesArg []reflect.Value
			for _, v := range args {
				valuesArg = append(valuesArg, reflect.ValueOf(v))
			}
			result := method.Call(valuesArg)

			results := make([]any, len(result))
			for i, v := range result {
				results[i] = v.Interface()
			}
			err, ok := results[len(result)-1].(error)
			if ok {
				rsp.Code = 500
				rsp.Msg = err.Error()
				conn.rspChan <- rsp
				return
			}
			rsp.Code = 200
			rsp.Data = results[0]
			conn.rspChan <- rsp
		}
	}
}

func (s *QueenTcpServer) writeHandle(conn *QueenTcpConn) {
	select {
	case rsp := <-conn.rspChan:
		defer conn.conn.Close()
		//发送数据
		err := conn.Send(rsp)
		if err != nil {
			log.Println(err)
		}

	}
}

func (s *QueenTcpServer) SetRegister(registerType string, option register.Option) {
	s.RegisterType = registerType
	s.RegisterOption = option
	if registerType == "nacos" {
		s.RegisterCli = &register.QueenNacosRegister{}
	}
	if registerType == "etcd" {
		s.RegisterCli = &register.QueenEtcdRegister{}
	}
}

func decodeFrame(conn net.Conn) (*QueenRpcMessage, error) {
	//1+1+4+1+1+1+8=17
	headers := make([]byte, 17)
	_, err := io.ReadFull(conn, headers)
	if err != nil {
		return nil, err
	}
	mn := headers[0]
	if mn != MagicNumber {
		return nil, errors.New("magic number error")
	}
	//version
	vs := headers[1]
	//full length
	//网络传输 大端
	fullLength := int32(binary.BigEndian.Uint32(headers[2:6]))
	//messageType
	messageType := headers[6]
	//压缩类型
	compressType := headers[7]
	//序列化类型
	seType := headers[8]
	//请求id
	requestId := int64(binary.BigEndian.Uint32(headers[9:]))

	msg := &QueenRpcMessage{
		Header: &Header{},
	}
	msg.Header.MagicNumber = mn
	msg.Header.Version = vs
	msg.Header.FullLength = fullLength
	msg.Header.MessageType = MessageType(messageType)
	msg.Header.CompressType = CompressType(compressType)
	msg.Header.SerializeType = SerializerType(seType)
	msg.Header.RequestId = requestId

	//body
	bodyLen := fullLength - 17
	body := make([]byte, bodyLen)
	_, err = io.ReadFull(conn, body)
	if err != nil {
		return nil, err
	}
	//编码的 先序列化 后 压缩
	//解码的时候 先解压缩，反序列化
	compress := loadCompress(CompressType(compressType))
	if compress == nil {
		return nil, errors.New("no compress")
	}
	body, err = compress.UnCompress(body)
	if compress == nil {
		return nil, err
	}
	serializer := loadSerializer(SerializerType(seType))
	if serializer == nil {
		return nil, errors.New("no serializer")
	}
	if MessageType(messageType) == msgRequest {
		if SerializerType(seType) == ProtoBuff {
			req := &Request{}
			err := serializer.DeSerialize(body, req)
			if err != nil {
				return nil, err
			}
			msg.Data = req
		} else {
			req := &QueenRpcRequest{}
			err := serializer.DeSerialize(body, req)
			if err != nil {
				return nil, err
			}
			msg.Data = req
		}
		return msg, nil
	}
	if MessageType(messageType) == msgResponse {
		if SerializerType(seType) == ProtoBuff {
			rsp := &Response{}
			err := serializer.DeSerialize(body, rsp)
			if err != nil {
				return nil, err
			}
			msg.Data = rsp
		} else {
			rsp := &QueenRpcResponse{}
			err := serializer.DeSerialize(body, rsp)
			if err != nil {
				return nil, err
			}
			msg.Data = rsp
		}

		return msg, nil
	}
	return nil, errors.New("no message type")
}

func loadSerializer(serializerType SerializerType) Serializer {
	switch serializerType {
	case Gob:
		return GobSerializer{}
	case ProtoBuff:
		return ProtobufSerializer{}
	}
	return nil
}

func loadCompress(compressType CompressType) CompressInterface {
	switch compressType {
	case Gzip:
		return GzipCompress{}
	}
	return nil
}

type QueenRpcClient interface {
	Connect() error
	Invoke(context context.Context, serviceName string, methodName string, args []any) (any, error)
	Close() error
}

type QueenTcpClient struct {
	conn        net.Conn
	option      TcpClientOption
	ServiceName string
	RegisterCli register.QueenRegister
}
type TcpClientOption struct {
	Retries           int
	ConnectionTimeout time.Duration
	SerializeType     SerializerType
	CompressType      CompressType
	Host              string
	Port              int
	RegisterType      string
	RegisterOption    register.Option
	RegisterCli       register.QueenRegister
}

var DefaultOption = TcpClientOption{
	Host:              "127.0.0.1",
	Port:              9222,
	Retries:           3,
	ConnectionTimeout: 5 * time.Second,
	SerializeType:     Gob,
	CompressType:      Gzip,
}

func NewTcpClient(option TcpClientOption) *QueenTcpClient {
	return &QueenTcpClient{option: option}
}

func (c *QueenTcpClient) Connect() error {
	var addr string
	err := c.RegisterCli.CreateCli(c.option.RegisterOption)
	if err != nil {
		panic(err)
	}
	addr, err = c.RegisterCli.GetValue(c.ServiceName)
	if err != nil {
		panic(err)
	}
	conn, err := net.DialTimeout("tcp", addr, c.option.ConnectionTimeout)
	if err != nil {
		return err
	}
	c.conn = conn
	return nil
}

func (c *QueenTcpClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

var reqId int64

func (c *QueenTcpClient) Invoke(ctx context.Context, serviceName string, methodName string, args []any) (any, error) {
	// 包装request对象，编码发送即可
	req := &QueenRpcRequest{}
	req.RequestId = atomic.AddInt64(&reqId, 1)
	req.ServiceName = serviceName
	req.MethodName = methodName
	req.Args = args

	headers := make([]byte, 17)

	headers[0] = MagicNumber //magic number

	headers[1] = Version //version
	// full length
	headers[6] = byte(msgRequest)                                  // 消息类型
	headers[7] = byte(c.option.CompressType)                       // 压缩类型
	headers[8] = byte(c.option.SerializeType)                      // 序列化
	binary.BigEndian.PutUint64(headers[9:], uint64(req.RequestId)) // 请求id
	serializer := loadSerializer(c.option.SerializeType)
	if serializer == nil {
		return nil, errors.New("no serializer")
	}
	var body []byte
	var err error
	if c.option.SerializeType == ProtoBuff {
		pReq := &Request{}
		pReq.RequestId = atomic.AddInt64(&reqId, 1)
		pReq.ServiceName = serviceName
		pReq.MethodName = methodName
		listValue, err := structpb.NewList(args)
		if err != nil {
			return nil, err
		}
		pReq.Args = listValue.Values
		body, err = serializer.Serialize(pReq)
	} else {
		body, err = serializer.Serialize(req)
	}

	if err != nil {
		return nil, err
	}
	compress := loadCompress(c.option.CompressType)
	if compress == nil {
		return nil, errors.New("no compress")
	}
	body, err = compress.Compress(body)
	if err != nil {
		return nil, err
	}
	fullLen := 17 + len(body)
	binary.BigEndian.PutUint32(headers[2:6], uint32(fullLen))

	_, err = c.conn.Write(headers[:])
	if err != nil {
		return nil, err
	}

	_, err = c.conn.Write(body[:])
	if err != nil {
		return nil, err
	}
	rspChan := make(chan *QueenRpcResponse)
	go c.readHandle(rspChan)
	rsp := <-rspChan
	return rsp, nil
}

func (c *QueenTcpClient) readHandle(rspChan chan *QueenRpcResponse) {
	defer func() {
		if err := recover(); err != nil {
			log.Println("QueenTcpClient readHandle recover: ", err)
			c.conn.Close()
		}
	}()
	for {
		msg, err := decodeFrame(c.conn)
		if err != nil {
			log.Println("未解析出任何数据")
			rsp := &QueenRpcResponse{}
			rsp.Code = 500
			rsp.Msg = err.Error()
			rspChan <- rsp
			return
		}
		if msg.Header.MessageType == msgResponse {
			if msg.Header.SerializeType == ProtoBuff {
				rsp := msg.Data.(*Response)
				asInterface := rsp.Data.AsInterface()
				marshal, _ := json.Marshal(asInterface)
				rsp1 := &QueenRpcResponse{}
				json.Unmarshal(marshal, rsp1)
				rspChan <- rsp1
			} else {
				rsp := msg.Data.(*QueenRpcResponse)
				rspChan <- rsp
			}
			return
		}
	}
}

func (c *QueenTcpClient) decodeFrame(conn net.Conn) (*QueenRpcMessage, error) {
	// 1+1+4+1+1+1+8=17
	headers := make([]byte, 17)
	_, err := io.ReadFull(conn, headers)
	if err != nil {
		return nil, err
	}
	mn := headers[0]
	if mn != MagicNumber {
		return nil, errors.New("magic number error")
	}
	vs := headers[1]                                           //version
	fullLength := int32(binary.BigEndian.Uint32(headers[2:6])) // full length
	messageType := headers[6]                                  // messageType
	compressType := headers[7]                                 // 压缩类型
	seType := headers[8]                                       // 序列化类型
	requestId := int64(binary.BigEndian.Uint32(headers[9:]))   // 请求id
	msg := &QueenRpcMessage{
		Header: &Header{},
	}
	msg.Header.MagicNumber = mn
	msg.Header.Version = vs
	msg.Header.FullLength = fullLength
	msg.Header.MessageType = MessageType(messageType)
	msg.Header.CompressType = CompressType(compressType)
	msg.Header.SerializeType = SerializerType(seType)
	msg.Header.RequestId = requestId

	bodyLen := fullLength - 17
	body := make([]byte, bodyLen)
	_, err = io.ReadFull(conn, body)
	if err != nil {
		return nil, err
	}
	// 编码时先序列化后压缩
	// 解码时先解压缩后反序列化
	compress := loadCompress(CompressType(compressType))
	if compress == nil {
		return nil, errors.New("no compress")
	}
	body, err = compress.UnCompress(body)
	if compress == nil {
		return nil, err
	}
	serializer := loadSerializer(SerializerType(seType))
	if serializer == nil {
		return nil, errors.New("no serializer")
	}
	if MessageType(messageType) == msgRequest {
		req := &QueenRpcRequest{}
		err := serializer.DeSerialize(body, req)
		if err != nil {
			return nil, err
		}
		msg.Data = req
		return msg, nil
	}
	if MessageType(messageType) == msgResponse {
		rsp := &QueenRpcResponse{}
		err := serializer.DeSerialize(body, rsp)
		if err != nil {
			return nil, err
		}
		msg.Data = rsp
		return msg, nil
	}
	return nil, errors.New("no message type")
}

type QueenTcpClientProxy struct {
	client *QueenTcpClient
	option TcpClientOption
}

func NewQueenTcpClientProxy(option TcpClientOption) *QueenTcpClientProxy {
	return &QueenTcpClientProxy{option: option}
}
func (p *QueenTcpClientProxy) Call(ctx context.Context, serviceName string, methodName string, args []any) (any, error) {
	client := NewTcpClient(p.option)
	client.ServiceName = serviceName
	if p.option.RegisterType == "nacos" {
		client.RegisterCli = &register.QueenNacosRegister{}
	}
	if p.option.RegisterType == "etcd" {
		client.RegisterCli = &register.QueenEtcdRegister{}
	}
	p.client = client
	err := client.Connect()
	if err != nil {
		return nil, err
	}
	for i := 0; i < p.option.Retries; i++ {
		result, err := client.Invoke(ctx, serviceName, methodName, args)
		if err != nil {
			if i >= p.option.Retries-1 {
				log.Println(errors.New("already retry all time"))
				client.Close()
				return nil, err
			}
			// 睡眠一小会
			continue
		}
		client.Close()
		return result, nil
	}
	return nil, errors.New("retry time is 0")
}
