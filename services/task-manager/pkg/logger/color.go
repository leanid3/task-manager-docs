package logger

// ANSI цветовые коды
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorGray   = "\033[37m"
	ColorWhite  = "\033[97m"
)

// ColorLevelMap для цветного вывода уровней
var ColorLevelMap = map[string]string{
	"DEBUG": ColorCyan,
	"INFO":  ColorGreen,
	"WARN":  ColorYellow,
	"ERROR": ColorRed,
}

// Цвета для категорий
var ColorCategoryMap = map[string]string{
	"app":    ColorGreen,
	"health": ColorBlue,
	"http":   ColorPurple,
	"kafka":  ColorYellow,
	"minio":  ColorCyan,
	"task":   ColorWhite,
	"s3":     ColorGray,
}
