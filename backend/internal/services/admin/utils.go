package admin

func questionInputNames(inputs []QuestionInput) []int {
	names := make([]int, 0, len(inputs))
	for _, q := range inputs {
		if q.Name != nil && *q.Name > 0 {
			names = append(names, *q.Name)
		}
	}
	return names
}
