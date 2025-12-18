### Создать Локализатор

Желательно инициализацию производить единожды на уровне приложения
```go
_, err := localizer.InitLocalizer()
```

В дальнейшем обращение к локалайзеру производить через импорт пакета в нужном месте

```go
import "gitlab.almanit.kz/jmart/gosdk/pkg/localizer"

m := localizer.Locale.GetMessageOne("key", "ru")
```

Для корректной работы требуется:
1. наличие директории `"/localization"`  в главной директории.
2. в директории  `"/localization"`  наличие файлов `en.json` `ru.json` (В случае необходимости добавления нового языка, добавляем файл, допустим `kk.json`)

<hr style="border: 1px solid orange;"/>

### Формат файла {lang}.json

1. в файлах  `en.json` `ru.json` нужно прописать текст в формате <code>key-valye</code> 
2. Например в файле  `ru.json` 
```json
{
  "Hello": "Салам брат",
  "Hello Users": "Салам {{.Name}}"
}
```

`"Hello"` -  является ключом, по этому ключу можно будет найти его перевод 

<hr style="border: 1px solid orange;"/>

### Чтобы найти все переводы по ключу вызываем метод
```go
value, err := l.GetMessage("Hello")
```

в value будет `map[string]string`. Например
```go
map[en:Hello brother ru:Салам брат]
```
<hr style="border: 1px solid orange;"/>

### Чтобы найти один перевод по ключу и определенному языку вызываем метод
```go
value := l.GetMessageOne("Hello",  constants.LangCodeRu)
```

в value будет `string`. 

<hr style="border: 1px solid orange;"/>

### Получить Динамичную строчку по всем переводам

```go
value, err := l.GetAllMessageWithTemplate("Hello Users", map[string]interface{}{
    "Name": "Aza",
})
```
в value будет `map[string]string`. Например
```go
map[en:Hello Aza ru:Салам Aza]
```

<hr style="border: 1px solid orange;"/>

### Получить Динамичную строчку по одному переводу

```go
value, err := l.GetMessageWithTemplate("Hello Users", constants.LangCodeRu, map[string]interface{}{
    "Name": "Aza",
})
```
в value будет `string`. 
<hr style="border: 1px solid orange;"/>


Примечание:

Используем эту библиотеку  https://github.com/nicksnyder/go-i18n/

