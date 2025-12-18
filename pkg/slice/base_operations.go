package slice

import "encoding/json"

// RemoveAtOrderly - удаляет элемент по индексу из слайса, сохраняя порядок. Если индек не корректный возвращает слайс без изменений
func RemoveAtOrderly[T any](slice []T, index int) []T {
	if index < 0 || index >= len(slice) {
		return slice
	}

	return append(slice[:index], slice[index+1:]...)
}

// RemoveAt - удаляет элемент по индексу из слайса без сохранения порядка. Если индек не корректный возвращает слайс без изменений
func RemoveAt[T any](slice []T, index int) []T {
	if index < 0 || index >= len(slice) {
		return slice
	}

	slice[index] = slice[len(slice)-1]

	return slice[:len(slice)-1]
}

// RemoveMultiple - удаляет элементы, соответствующие условию фильтрации.
func RemoveMultiple[T any](slice []T, filter func(T) bool) []T {
	newSlice := make([]T, 0, len(slice))

	for _, val := range slice {
		if !filter(val) {
			newSlice = append(newSlice, val)
		}
	}

	return newSlice
}

// Filter - Фильтрация слайса
func Filter[T any](slice []T, filter func(T) bool) []T {
	newSlice := make([]T, 0, len(slice))

	for _, val := range slice {
		if filter(val) {
			newSlice = append(newSlice, val)
		}
	}

	return newSlice
}

// Map - Созание нового слайса
func Map[T any, Y any](slice []T, callback func(int, T) Y) []Y {
	newSlice := make([]Y, len(slice))

	for i, val := range slice {
		newSlice[i] = callback(i, val)
	}

	return newSlice
}

// StructToMap - Превращает структуру в map[string]interface{} используя json
func StructToMap(obj interface{}) (newMap map[string]interface{}, err error) {
	data, err := json.Marshal(obj)

	if err != nil {
		return
	}

	err = json.Unmarshal(data, &newMap)

	return
}

func RemoveDuplicates[T comparable](input []T) []T {
	m := map[T]bool{}
	var result []T
	for _, v := range input {
		if !m[v] {
			m[v] = true
			result = append(result, v)
		}
	}
	return result
}
