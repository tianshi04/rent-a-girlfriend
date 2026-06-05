package vo

// ContextKey đại diện cho kiểu dữ liệu tự định nghĩa cho khóa trong context.
type ContextKey string

// TxKey là khóa dùng để lưu trữ và truy vấn database transaction trong context.
const TxKey ContextKey = "tx"
