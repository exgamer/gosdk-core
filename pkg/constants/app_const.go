package constants

// AppInfoKey Строковый ключ AppInfo в контексте до v1.1.0. Ключом контекста core больше не служит.
//
// Deprecated: класть AppInfo — context.WithAppInfo, читать — context.GetAppInfoFromContext.
const AppInfoKey string = "app_info"
