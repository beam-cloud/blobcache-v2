// Code generated by the FlatBuffers compiler. DO NOT EDIT.

package blobcache_fbs

import (
	flatbuffers "github.com/google/flatbuffers/go"
)

type GetStateRequestT struct {
}

func (t *GetStateRequestT) Pack(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	if t == nil {
		return 0
	}
	GetStateRequestStart(builder)
	return GetStateRequestEnd(builder)
}

func (rcv *GetStateRequest) UnPackTo(t *GetStateRequestT) {
}

func (rcv *GetStateRequest) UnPack() *GetStateRequestT {
	if rcv == nil {
		return nil
	}
	t := &GetStateRequestT{}
	rcv.UnPackTo(t)
	return t
}

type GetStateRequest struct {
	_tab flatbuffers.Table
}

func GetRootAsGetStateRequest(buf []byte, offset flatbuffers.UOffsetT) *GetStateRequest {
	n := flatbuffers.GetUOffsetT(buf[offset:])
	x := &GetStateRequest{}
	x.Init(buf, n+offset)
	return x
}

func FinishGetStateRequestBuffer(builder *flatbuffers.Builder, offset flatbuffers.UOffsetT) {
	builder.Finish(offset)
}

func GetSizePrefixedRootAsGetStateRequest(buf []byte, offset flatbuffers.UOffsetT) *GetStateRequest {
	n := flatbuffers.GetUOffsetT(buf[offset+flatbuffers.SizeUint32:])
	x := &GetStateRequest{}
	x.Init(buf, n+offset+flatbuffers.SizeUint32)
	return x
}

func FinishSizePrefixedGetStateRequestBuffer(builder *flatbuffers.Builder, offset flatbuffers.UOffsetT) {
	builder.FinishSizePrefixed(offset)
}

func (rcv *GetStateRequest) Init(buf []byte, i flatbuffers.UOffsetT) {
	rcv._tab.Bytes = buf
	rcv._tab.Pos = i
}

func (rcv *GetStateRequest) Table() flatbuffers.Table {
	return rcv._tab
}

func GetStateRequestStart(builder *flatbuffers.Builder) {
	builder.StartObject(0)
}
func GetStateRequestEnd(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	return builder.EndObject()
}
