package animxmaker

import (
	"bytes"
	"encoding/binary"
	"errors"
)

func write7BitEncodedInt(n int) []byte {
	var result []byte
	for {
		// Get the least significant 7 bits
		byteVal := byte(n & 0x7F)
		n >>= 7 // Shift right 7 bits to process the next chunk

		if n != 0 {
			// More bytes to come: set MSB to 1
			byteVal |= 0x80
		}

		// Append this byte to the result
		result = append(result, byteVal)

		if n == 0 {
			break
		}
	}
	return result
}

func encodeAnimString(s string, isValue bool) ([]byte) {
	var buf bytes.Buffer
	if isValue {
		buf.WriteByte(1)
	}
	// Encode length
	binary.Write(&buf, binary.LittleEndian, write7BitEncodedInt(len(s)))
	buf.Write([]byte(s))
	return buf.Bytes()
}

type KeyFrame[T any] struct {
	Position float32
	Value T 
}

func (a *KeyFrame[T]) EncodeKeyframe() ([]byte, error){
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, a.Position)
	switch v := any(a.Value).(type) {
			case string:
				buf.Write(encodeAnimString(v, true))
			case int32:
				binary.Write(&buf, binary.LittleEndian, v)
			case int:
				v32 := int32(v)
				binary.Write(&buf, binary.LittleEndian, v32)
			case float32:
				binary.Write(&buf, binary.LittleEndian, v)
			default:
				return nil, errors.New("Unsupported type")
			// pain
		}		
	return buf.Bytes(), nil
}

type AnimationTrackWrapper interface {
	EncodeTrack() ([]byte, error)
	GetTrackDuration() float32
}

type AnimationTrack[T any] struct {
	Node string
	Property string
	Keyframes []KeyFrame[T]
}

func (a *AnimationTrack[T]) GetTrackDuration() float32 {
	if len(a.Keyframes) == 0 {
		return 0
	}
	return a.Keyframes[len(a.Keyframes)-1].Position
}

func (a *AnimationTrack[T]) EncodeTrack() ([]byte, error){

	var buf bytes.Buffer
	buf.WriteByte(1) // track type
	if len(a.Keyframes) == 0 {
		// If there's no keyframes we need to write a dummy keyframe in order to prevent resontie from crashing
		buf.WriteByte(39) // 39 = String
		buf.Write(encodeAnimString(a.Node, false))
		buf.Write(encodeAnimString(a.Property, false))
		binary.Write(&buf, binary.LittleEndian, write7BitEncodedInt(1)) // keyframe count
		keyframe := KeyFrame[string]{
			Position: 0,
			Value: "No results",
		}
		keyframeBytes, err := keyframe.EncodeKeyframe()
		if err != nil {
			return nil, err
		}
		buf.Write(keyframeBytes)
		return buf.Bytes(), nil
	}
	switch any(a.Keyframes[0].Value).(type) {
		case string:
			buf.WriteByte(39) // 39 = String
		case int32:
			buf.WriteByte(10) // 10 = Int
		case int:
			buf.WriteByte(10) // 10 = Int
		case float32:
			buf.WriteByte(21) // 21 = Float
		default:
			return nil, errors.New("Unsupported type")

		// pain again
	}
	buf.Write(encodeAnimString(a.Node, false))
	buf.Write(encodeAnimString(a.Property, false))
	binary.Write(&buf, binary.LittleEndian, write7BitEncodedInt(len(a.Keyframes))) // keyframe count
	for _, keyframe := range a.Keyframes{
		var keyframeBytes []byte
		keyframeBytes, err := keyframe.EncodeKeyframe()
		if err != nil {
			return nil, err
		}
		buf.Write(keyframeBytes)
	}
	return buf.Bytes(), nil
}

type Animation struct {
	Tracks []AnimationTrackWrapper
}

func (a *Animation) EncodeAnimation(animationName string) ([]byte, error) {
	var buf bytes.Buffer

	buf.Write(encodeAnimString("AnimX", false))             // "Magic Word"
	binary.Write(&buf, binary.LittleEndian, int32(1))	// Version
	buf.Write(write7BitEncodedInt(len(a.Tracks)))           // Track count
	
	var maxDuration float32 = 0
	for _, track := range a.Tracks {
		if track.GetTrackDuration() > maxDuration {
			maxDuration = track.GetTrackDuration()
		}
	}
	binary.Write(&buf, binary.LittleEndian, maxDuration) 	// Max duration
	buf.Write(encodeAnimString(animationName, false))     	// Animation name
	buf.WriteByte(0)                     			// Encoding type (0 is default binary reader)
	for _, track := range a.Tracks {
		trackBytes, err := track.EncodeTrack()
		if err != nil {
			return nil, err
		}
		buf.Write(trackBytes)
	}
	
	return buf.Bytes(), nil
}
