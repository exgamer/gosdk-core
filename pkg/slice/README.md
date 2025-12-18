# Slice Utilities

Этот пакет предоставляет полезные функции для работы с слайсами в Go.

## Функции

### RemoveAtOrderly
Удаляет элемент по индексу, сохраняя порядок элементов.

**Пример использования:**
```go
slice := []int{1, 2, 3, 4, 5}
slice = RemoveAtOrderly(slice, 2)
fmt.Println(slice) // [1, 2, 4, 5]
```

### RemoveAt
Удаляет элемент по индексу без сохранения порядка (быстрое удаление).

**Пример использования:**
```go
slice := []int{1, 2, 3, 4, 5}
slice = RemoveAt(slice, 2)
fmt.Println(slice) // [1, 2, 5, 4]
```

### RemoveMultiple
Удаляет элементы, соответствующие заданному фильтру.

**Пример использования:**
```go
slice := []int{1, 2, 3, 4, 5}
slice = RemoveMultiple(slice, func(val int) bool {
    return val%2 == 0 // удаляет все четные числа
})
fmt.Println(slice) // [1, 3, 5]
```

### Filter
Фильтрует слайс, оставляя только элементы, соответствующие условию.

**Пример использования:**
```go
slice := []string{"apple", "banana", "cherry"}
slice = Filter(slice, func(val string) bool {
    return val == "banana"
})
fmt.Println(slice) // ["banana"]
```

### Map
Создает новый слайс, применяя функцию к каждому элементу исходного слайса.

**Пример использования:**
```go
slice := []int{1, 2, 3}
squared := Map(slice, func(i int, val int) int {
    return val * val
})
fmt.Println(squared) // [1, 4, 9]
```

### StructToMap
Конвертирует структуру в `map[string]interface{}` с использованием JSON.

**Пример использования:**
```go
type User struct {
    Name string
    Age  int
}

user := User{Name: "John", Age: 30}
userMap, err := StructToMap(user)
if err != nil {
    fmt.Println("Ошибка:", err)
} else {
    fmt.Println(userMap) // map[Name:John Age:30]
}
```

