package animxmaker

func ListTrack[T any](list []T, nodeName string, propertyName string) AnimationTrackWrapper{
	var keyframes []KeyFrame[T]
	for i, item := range list {
		keyframe := KeyFrame[T]{
			Position: float32(i),
			Value: item,
		}
		keyframes = append(keyframes, keyframe)
	}
	track := AnimationTrack[T]{
		Keyframes: keyframes,
		Node: nodeName,
		Property: propertyName,
	}
 
	return AnimationTrackWrapper(&track)
}
