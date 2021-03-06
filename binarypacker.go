//
// packager crypto package
// Author: Guido Ronchetti <guido.ronchetti@nexo.cloud>
// v1.0 14/07/2017
//

package binarypacker

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"
	"unsafe"
)

const (
	kDYStringType    = iota
	kDYDataType      = iota
	kDYIntegerType   = iota
	kDYRealType      = iota
	kDYBooleanType   = iota
	kDYNullType      = iota
	kDYNumericStruct = iota
	kDYArrayType     = iota
)

type binaryType int32

type binaryPacket struct {
	binaryType
	size int64
	data []byte
}

func (b *binaryPacket) encode() ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, b.binaryType)
	if err != nil {
		return nil, err
	}
	err = binary.Write(buf, binary.LittleEndian, b.size)
	if err != nil {
		return nil, err
	}
	_, err = buf.Write(b.data)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func decode(data []byte) (*binaryPacket, error) {
	pack := &binaryPacket{}
	minSize := (int(unsafe.Sizeof(pack.binaryType)) +
		int(unsafe.Sizeof(pack.size)+1))
	if len(data) < minSize {
		return nil, fmt.Errorf(
			"data size is less than required, having %d expecting at least %d",
			len(data),
			minSize,
		)
	}
	buf := bytes.NewReader(data)
	err := binary.Read(buf, binary.LittleEndian, &pack.binaryType)
	if err != nil {
		return nil, err
	}
	err = binary.Read(buf, binary.LittleEndian, &pack.size)
	if err != nil {
		return nil, err
	}
	if pack.size > int64(len(data)) {
		return nil, fmt.Errorf(
			"Invalid readed size %d > than encoded size %d",
			pack.size, len(data),
		)
	}
	pack.data = make([]byte, pack.size)
	var idx int64
	for idx = 0; idx < pack.size; idx++ {
		pack.data[idx], err = buf.ReadByte()
		if err != nil {
			return nil, err
		}
	}
	return pack, nil
}

func getPacketMetadata(data []byte) (binaryType, int64, error) {
	pack := &binaryPacket{}
	minSize := (int(unsafe.Sizeof(pack.binaryType)) +
		int(unsafe.Sizeof(pack.size)+1))
	if len(data) < minSize {
		return 0, 0, fmt.Errorf(
			"data size is less than required, having %d expecting at least %d",
			len(data),
			minSize,
		)
	}
	buf := bytes.NewReader(data)
	err := binary.Read(buf, binary.LittleEndian, &pack.binaryType)
	if err != nil {
		return 0, 0, err
	}
	err = binary.Read(buf, binary.LittleEndian, &pack.size)
	if err != nil {
		return 0, 0, err
	}
	return pack.binaryType, pack.size, nil
}

func marshalNil() ([]byte, error) {
	pack := &binaryPacket{
		binaryType: kDYNullType,
		size:       1,
		data:       []byte{0},
	}
	return pack.encode()
}

func (p *binaryPacket) unmarshalNil() (interface{}, error) {
	return nil, nil
}

func marshalArray(value []interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	for _, element := range value {
		data, err := Marshal(element)
		if err != nil {
			return nil, err
		}
		_, err = buf.Write(data)
		if err != nil {
			return nil, err
		}
	}
	pack := &binaryPacket{
		binaryType: kDYArrayType,
		size:       int64(buf.Len()),
		data:       buf.Bytes(),
	}
	return pack.encode()
}

func marshalStringArray(value []string) ([]byte, error) {
	converted := make([]interface{}, len(value))
	for idx, element := range value {
		converted[idx] = element
	}
	return marshalArray(converted)
}

func marshalIntegerArray(value []int64) ([]byte, error) {
	converted := make([]interface{}, len(value))
	for idx, element := range value {
		converted[idx] = element
	}
	return marshalArray(converted)
}

func marshalFloatArray(value []float64) ([]byte, error) {
	converted := make([]interface{}, len(value))
	for idx, element := range value {
		converted[idx] = element
	}
	return marshalArray(converted)
}

func marshalBoolArray(value []bool) ([]byte, error) {
	converted := make([]interface{}, len(value))
	for idx, element := range value {
		converted[idx] = element
	}
	return marshalArray(converted)
}

func (p *binaryPacket) unmarshalArray() ([]interface{}, error) {
	var readed int64
	result := make([]interface{}, 0)
	for readed < p.size {
		decoded, err := Unmarshal(p.data[readed:])
		if err != nil {
			return nil, err
		}
		_, size, err := getPacketMetadata(p.data[readed:])
		if err != nil {
			return nil, err
		}
		result = append(result, decoded)
		readed += (size +
			int64(unsafe.Sizeof(p.binaryType)) +
			int64(unsafe.Sizeof(p.size)))
	}
	return result, nil
}

func marshalString(value string) ([]byte, error) {
	binary := make([]byte, 0)
	binary = append(binary, []byte(value)...)
	binary = append(binary, byte(0))
	pack := &binaryPacket{
		binaryType: kDYStringType,
		size:       int64(len(value) + 1),
		data:       binary,
	}
	return pack.encode()
}

func (p *binaryPacket) unmarshalString() (string, error) {
	return string(p.data[:p.size-1]), nil
}

func marshalFloat(value float64) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, value)
	if err != nil {
		return nil, err
	}
	pack := &binaryPacket{
		binaryType: kDYRealType,
		size:       int64(buf.Len()),
		data:       buf.Bytes(),
	}
	return pack.encode()
}

func (p *binaryPacket) unmarshalFloat() (float64, error) {
	var result float64
	buf := bytes.NewReader(p.data)
	err := binary.Read(buf, binary.LittleEndian, &result)
	if err != nil {
		return 0.0, err
	}
	return result, nil
}

func marshalInteger(value int64) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, value)
	if err != nil {
		return nil, err
	}
	pack := &binaryPacket{
		binaryType: kDYIntegerType,
		size:       int64(buf.Len()),
		data:       buf.Bytes(),
	}
	return pack.encode()
}

func (p *binaryPacket) unmarshalInteger() (int64, error) {
	var result int64
	buf := bytes.NewReader(p.data)
	err := binary.Read(buf, binary.LittleEndian, &result)
	if err != nil {
		return 0, err
	}
	return result, nil
}

func marshalBool(value bool) ([]byte, error) {
	var flag uint8 = 0
	if value == true {
		flag = 1
	}
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, flag)
	if err != nil {
		return nil, err
	}
	pack := &binaryPacket{
		binaryType: kDYBooleanType,
		size:       int64(buf.Len()),
		data:       buf.Bytes(),
	}
	return pack.encode()
}

func (p *binaryPacket) unmarshalBool() (bool, error) {
	var flag uint8
	buf := bytes.NewReader(p.data)
	err := binary.Read(buf, binary.LittleEndian, &flag)
	if err != nil {
		return false, err
	}
	if flag != 0 {
		return true, nil
	}
	return false, nil
}

func marshalData(value []byte) ([]byte, error) {
	pack := &binaryPacket{
		binaryType: kDYDataType,
		size:       int64(len(value)),
		data:       value,
	}
	return pack.encode()
}

func (p *binaryPacket) unmarshalData() ([]byte, error) {
	return p.data, nil
}

func Marshal(value interface{}) ([]byte, error) {
	if value == nil {
		return marshalNil()
	}
	switch value.(type) {
	case string:
		return marshalString(value.(string))
	case int:
		return marshalInteger(int64(value.(int)))
	case int32:
		return marshalInteger(int64(value.(int32)))
	case int64:
		return marshalInteger(value.(int64))
	case float64:
		return marshalFloat(value.(float64))
	case []byte:
		return marshalData(value.([]byte))
	case []string:
		return marshalStringArray(value.([]string))
	case []int64:
		return marshalIntegerArray(value.([]int64))
	case []float64:
		return marshalFloatArray(value.([]float64))
	case []bool:
		return marshalBoolArray(value.([]bool))
	case []interface{}:
		return marshalArray(value.([]interface{}))
	case bool:
		return marshalBool(value.(bool))
	}
	return nil, fmt.Errorf(
		"unsupported type unable to encode: %s",
		reflect.TypeOf(value),
	)
}

func Unmarshal(encoded []byte) (interface{}, error) {
	pack, err := decode(encoded)
	if err != nil {
		return nil, err
	}
	if pack.size == 0 {
		return nil, fmt.Errorf(
			"unsupported empty packet, unable to decode",
		)
	}
	switch pack.binaryType {
	case kDYStringType:
		return pack.unmarshalString()
	case kDYIntegerType:
		return pack.unmarshalInteger()
	case kDYDataType:
		return pack.unmarshalData()
	case kDYNullType:
		return pack.unmarshalNil()
	case kDYRealType:
		return pack.unmarshalFloat()
	case kDYArrayType:
		return pack.unmarshalArray()
	case kDYBooleanType:
		return pack.unmarshalBool()
	}
	return nil, fmt.Errorf(
		"unsupported type: %d",
		pack.binaryType,
	)
}
