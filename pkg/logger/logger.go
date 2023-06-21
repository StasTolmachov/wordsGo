package logger

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

var counter int
var mutex = &sync.Mutex{}

type CustomFormatter struct{}

func (f *CustomFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	mutex.Lock()
	counter++
	mutex.Unlock()

	timestampDate := fmt.Sprintf("%v", entry.Time.Format("2006-01-02"))
	timestampTime := "\033[33m" + fmt.Sprintf("%v", entry.Time.Format("15:04:05")) + "\033[0m" // Желтый цвет
	timestamp := timestampDate + " " + timestampTime

	level := strings.ToUpper(entry.Level.String())

	// Выводим уровень INFO синим цветом
	if level == "INFO" {
		level = "\033[34m" + level + "\033[0m"
	}

	message := "Message: " + "\033[32m" + entry.Message + "\033[0m" // Зеленый цвет

	pc, _, line, _ := runtime.Caller(9)
	function := fmt.Sprintf("Func: [%s:%d]", runtime.FuncForPC(pc).Name(), line) // Оборачиваем функцию в квадратные скобки и добавляем номер строки

	// Изменяем порядок вывода элементов в логе и добавляем название функции
	// Удаляем путь к файлу из вывода
	return []byte(fmt.Sprintf("%-30s %-15s %-5d %-40s %s\n", timestamp, level, counter, function, message)), nil
}

func LogSetupFile() {
	logFile, err := os.OpenFile("words.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		logrus.SetOutput(logFile)
	} else {
		logrus.Infof("Failed to log to file, using default stderr: %v", err)

	}

	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetReportCaller(true)
	logrus.SetFormatter(new(CustomFormatter))
}
func LogSetupConsole() {

	logrus.SetOutput(os.Stdout)

	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetReportCaller(true)
	logrus.SetFormatter(new(CustomFormatter))
}
