// Code generated by the FlatBuffers compiler. DO NOT EDIT.

package blobcache_fbs

import (
	flatbuffers "github.com/google/flatbuffers/go"
)

type StoreContentFromSourceRequestT struct {
	SourcePath string `json:"source_path"`
	SourceOffset int64 `json:"source_offset"`
}

func (t *StoreContentFromSourceRequestT) Pack(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	if t == nil {
		return 0
	}
	sourcePathOffset := flatbuffers.UOffsetT(0)
	if t.SourcePath != "" {
		sourcePathOffset = builder.CreateString(t.SourcePath)
	}
	StoreContentFromSourceRequestStart(builder)
	StoreContentFromSourceRequestAddSourcePath(builder, sourcePathOffset)
	StoreContentFromSourceRequestAddSourceOffset(builder, t.SourceOffset)
	return StoreContentFromSourceRequestEnd(builder)
}

func (rcv *StoreContentFromSourceRequest) UnPackTo(t *StoreContentFromSourceRequestT) {
	t.SourcePath = string(rcv.SourcePath())
	t.SourceOffset = rcv.SourceOffset()
}

func (rcv *StoreContentFromSourceRequest) UnPack() *StoreContentFromSourceRequestT {
	if rcv == nil {
		return nil
	}
	t := &StoreContentFromSourceRequestT{}
	rcv.UnPackTo(t)
	return t
}

type StoreContentFromSourceRequest struct {
	_tab flatbuffers.Table
}

func GetRootAsStoreContentFromSourceRequest(buf []byte, offset flatbuffers.UOffsetT) *StoreContentFromSourceRequest {
	n := flatbuffers.GetUOffsetT(buf[offset:])
	x := &StoreContentFromSourceRequest{}
	x.Init(buf, n+offset)
	return x
}

func FinishStoreContentFromSourceRequestBuffer(builder *flatbuffers.Builder, offset flatbuffers.UOffsetT) {
	builder.Finish(offset)
}

func GetSizePrefixedRootAsStoreContentFromSourceRequest(buf []byte, offset flatbuffers.UOffsetT) *StoreContentFromSourceRequest {
	n := flatbuffers.GetUOffsetT(buf[offset+flatbuffers.SizeUint32:])
	x := &StoreContentFromSourceRequest{}
	x.Init(buf, n+offset+flatbuffers.SizeUint32)
	return x
}

func FinishSizePrefixedStoreContentFromSourceRequestBuffer(builder *flatbuffers.Builder, offset flatbuffers.UOffsetT) {
	builder.FinishSizePrefixed(offset)
}

func (rcv *StoreContentFromSourceRequest) Init(buf []byte, i flatbuffers.UOffsetT) {
	rcv._tab.Bytes = buf
	rcv._tab.Pos = i
}

func (rcv *StoreContentFromSourceRequest) Table() flatbuffers.Table {
	return rcv._tab
}

func (rcv *StoreContentFromSourceRequest) SourcePath() []byte {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(4))
	if o != 0 {
		return rcv._tab.ByteVector(o + rcv._tab.Pos)
	}
	return nil
}

func (rcv *StoreContentFromSourceRequest) SourceOffset() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(6))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *StoreContentFromSourceRequest) MutateSourceOffset(n int64) bool {
	return rcv._tab.MutateInt64Slot(6, n)
}

func StoreContentFromSourceRequestStart(builder *flatbuffers.Builder) {
	builder.StartObject(2)
}
func StoreContentFromSourceRequestAddSourcePath(builder *flatbuffers.Builder, sourcePath flatbuffers.UOffsetT) {
	builder.PrependUOffsetTSlot(0, flatbuffers.UOffsetT(sourcePath), 0)
}
func StoreContentFromSourceRequestAddSourceOffset(builder *flatbuffers.Builder, sourceOffset int64) {
	builder.PrependInt64Slot(1, sourceOffset, 0)
}
func StoreContentFromSourceRequestEnd(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	return builder.EndObject()
}
