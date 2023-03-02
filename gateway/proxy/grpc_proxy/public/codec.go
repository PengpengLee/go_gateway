package public

import (
	"fmt"
	"google.golang.org/grpc/encoding"

	"github.com/golang/protobuf/proto"
)

// 自定义编码解码器
// 支持proto消息、原始数组
//
// 使用方法：
//	1.服务端注册，确保init函数执行，将自定义编码注册到服务端系统中
//	2.客户端注册：
//		response, err := myclient.MyCall(ctx, request, grpc.CallContentSubtype("mycodec"))
//		myclient := grpc.Dial(ctx, target, grpc.WithDefaultCallOptions(grpc.CallContentSubtype("mycodec")))

// init 初始化函数，注册自定义编码
func init() {
	encoding.RegisterCodec(Codec())
}

// Codec returns a proxying encoding.Codec with the default protobuf codec as parent.
//
// See CodecWithParent.
// 构建输出函数
func Codec() encoding.Codec {
	return CodecWithParent(&protoCodec{})
}

// CodecWithParent returns a proxying encoding.Codec with a user provided codec as parent.
//
// This codec is *crucial* to the functioning of the proxy. It allows the proxy server to be oblivious
// to the schema of the forwarded messages. It basically treats a gRPC message Frame as raw bytes.
// However, if the server handler, or the client caller are not proxy-internal functions it will fall back
// to trying to decode the message using a fallback codec.
func CodecWithParent(fallback encoding.Codec) encoding.Codec {
	return &rawCodec{fallback}
}

type rawCodec struct {
	parentCodec encoding.Codec
}

// Frame 帧，二进制数据
type Frame struct {
	payload []byte
}

// 构建原始字节解码器
func (c *rawCodec) Marshal(v interface{}) ([]byte, error) {
	out, ok := v.(*Frame)
	if !ok {
		return c.parentCodec.Marshal(v)
	}
	return out.payload, nil

}

func (c *rawCodec) Unmarshal(data []byte, v interface{}) error {
	dst, ok := v.(*Frame)
	if !ok {
		return c.parentCodec.Unmarshal(data, v)
	}
	dst.payload = data
	return nil
}

func (c *rawCodec) Name() string {
	return fmt.Sprintf("proxy>%s", c.parentCodec.Name())
}

// protoCodec is a Codec implementation with protobuf. It is the default rawCodec for gRPC.
// 构建proto解码器
type protoCodec struct{}

func (protoCodec) Marshal(v interface{}) ([]byte, error) {
	return proto.Marshal(v.(proto.Message))
}

func (protoCodec) Unmarshal(data []byte, v interface{}) error {
	return proto.Unmarshal(data, v.(proto.Message))
}

func (protoCodec) Name() string {
	return "proto"
}
