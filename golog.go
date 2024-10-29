package golog

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

type GoLog struct {
	ctx         context.Context
	jobsChannel chan Log
	doneChannel chan struct{}
	config      Config
	logs        []Log
}

func (l *GoLog) handleLogs() {

	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	for {
		select {

		case <-l.ctx.Done():
			fmt.Println("golog context done")
			return

		case <-ticker.C:

			if len(l.logs) > 0 {

				go l.writeServer(l.logs)
				l.logs = make([]Log, 0)
			}

		case <-l.doneChannel:
			fmt.Println("golog done channel received")
			return

		case record := <-l.jobsChannel:

			if l.config.LogStdout {

				go l.writeStdout(record)
			}

			if l.config.LogFile {

				go l.writeFile(record)
			}

			if l.config.LogServer {

				l.logs = append(l.logs, record)
			}
		}
	}
}

func (l *GoLog) writeStdout(record Log) {

	if record.Data != nil {

		fmt.Printf("%s [%s] %s - %s, data: %v \n", time.Now().Format("2006/01/02 15:04:05"), record.Level.String(), record.Title, record.Message, record.Data)
		return
	} else {

		fmt.Printf("%s [%s] %s - %s \n", time.Now().Format("2006/01/02 15:04:05"), record.Level.String(), record.Title, record.Message)
	}
}

func (l *GoLog) writeFile(record Log) {

	if record.Data != nil {

		log.Printf("[%s] %s - %s, data: %v \n", record.Level.String(), record.Title, record.Message, record.Data)
		return
	} else {

		log.Printf("[%s] %s - %s \n", record.Level.String(), record.Title, record.Message)
	}
}

func (l *GoLog) writeServer(records []Log) {

	var logs []HttpLog
	for _, record := range records {

		logs = append(logs, record.ToHttpLog())
	}

	request := LogsRequest{
		Logs: logs,
	}

	data, err := json.Marshal(request)
	if err != nil {

		fmt.Println("error marshalling logs")
		return
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s://%s:%s/log/create-many", l.config.ServerApiProtocol, l.config.ServerApiHost, l.config.ServerApiPort), bytes.NewBuffer(data))
	if err != nil {

		fmt.Println("error creating request", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", l.generateHash())
	req.Header.Set("Platform-Name", l.config.ServerPlatfrom)

	// Disable server certificate verification
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	resp, err := client.Do(req)
	if err != nil {

		fmt.Println("error sending logs, err: ", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {

			fmt.Println("error reading response body", err)
			return
		}

		fmt.Println("error sending logs status code: ", resp.StatusCode, " body: ", string(body))
		return
	}
}

// SetConfig is used to set the logger configuration
func (l *GoLog) SetConfig(config Config) {

	l.config = config

	log.SetOutput(&lumberjack.Logger{
		Filename:   config.FileLocation,
		MaxSize:    l.config.FileMaxSize,
		MaxBackups: l.config.FileMaxBackups,
	})
}

// Log is used to log a message
func (l *GoLog) Log(level Level, title string, message string, data ...any) {

	go func(level Level, title string, message string, data ...any) {

		l.jobsChannel <- Log{
			Enviroment: l.config.Enviroment,
			Level:      level,
			Title:      title,
			Message:    message,
			Data:       data,
		}
	}(level, title, message, data...)
}

// Trace is used to log a trace message
func (l *GoLog) Trace(title string, message string, data ...any) {

	l.Log(Trace, title, message, data...)
}

// Debug is used to log a debug message
func (l *GoLog) Debug(title string, message string, data ...any) {

	l.Log(Debug, title, message, data...)
}

// Info is used to log an info message
func (l *GoLog) Info(title string, message string, data ...any) {

	l.Log(Info, title, message, data...)
}

// Warn is used to log a warning message
func (l *GoLog) Warn(title string, message string, data ...any) {

	l.Log(Warn, title, message, data...)
}

// Error is used to log an error message
func (l *GoLog) Error(title string, message string, data ...any) {

	l.Log(Error, title, message, data...)
}

// Fatal is used to log a fatal message
func (l *GoLog) Fatal(title string, message string, data ...any) {

	l.Log(Fatal, title, message, data...)
}

// Panic is used to log a panic message
func (l *GoLog) Panic(title string, message string, data ...any) {

	l.Log(Panic, title, message, data...)
}

// Done is used to close the logger and send the logs to the server
func (l *GoLog) Done() {

	l.doneChannel <- struct{}{}
	time.Sleep(1 * time.Second)
	l.writeServer(l.logs)
	l.logs = make([]Log, 0)
	close(l.jobsChannel)
	close(l.doneChannel)
}

// NewGoLog is used to create a new logger
func NewGoLog(ctx context.Context, config Config) *GoLog {

	GoLog := &GoLog{
		ctx:         ctx,
		jobsChannel: make(chan Log, 1000),
		doneChannel: make(chan struct{}),
		config:      config,
		logs:        make([]Log, 0),
	}

	go GoLog.handleLogs()

	return GoLog
}
