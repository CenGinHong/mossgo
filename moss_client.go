// @Author: 陈健航
// @Date: 2021/1/11 20:07
// @Description:
package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/url"
	"strings"
)

type Stage int

const (
	disconnected Stage = iota
	awaitingInitialization
	awaitingLanguage
	awaitingFiles
	awaitingQuery
	awaitingResults
	awaitingEnd
)

type MossSocketClient struct {
	currentStage       Stage
	addr               string
	userID             string
	language           string
	setID              int
	optM               int64
	optD               int
	optX               int
	optN               int
	optC               string
	ResultURL          *url.URL
	supportedLanguages []string
	conn               *net.TCPConn
}

// NewMossSocketClient 构造函数
// @params language 语言
// @params userId 用户id
// @return *MossSocketClient
// @return error
// @date 2021-01-11 17:03:26
func NewMossSocketClient(language, userId string) (*MossSocketClient, error) {

	supportedLanguages := []string{"c", "cc", "java", "ml", "pascal", "ada", "lisp", "schema", "haskell", "fortran",
		"ascii", "vhdl", "perl", "matlab", "python", "mips", "prolog", "spice", "vb", "csharp", "modula2", "a8086",
		"javascript", "plsql"}
	isContains := false
	for _, v := range supportedLanguages {
		if v == language {
			isContains = true
			break
		}
	}
	if isContains {
		return &MossSocketClient{
			currentStage:       disconnected,
			setID:              1,
			optM:               10,
			optD:               1,
			optX:               0,
			optN:               250,
			optC:               "",
			supportedLanguages: supportedLanguages,
			addr:               "moss.stanford.edu:7690",
			userID:             userId,
			language:           language,
		}, nil
	} else {
		return nil, errors.New("MOSS Server does not recognize this programming language")
	}
}

// Close 关闭
// @receiver c
// @return error
// @date 2021-01-11 17:04:12
func (c *MossSocketClient) Close() error {
	defer func() {
		c.currentStage = disconnected
	}()
	if err := c.sendCommand("end\n"); err != nil {
		return err
	}
	if err := c.conn.Close(); err != nil {
		return err
	}
	return nil
}

func (c *MossSocketClient) connect() error {
	if c.currentStage != disconnected {
		return errors.New("client is already connected")
	} else {
		tcpAddr, err := net.ResolveTCPAddr("tcp", c.addr)
		if err != nil {
			return err
		}
		conn, err := net.DialTCP("tcp", nil, tcpAddr)
		if err != nil {
			return err
		}
		c.conn = conn
		if err = c.conn.SetKeepAlive(true); err != nil {
			return err
		}
		c.currentStage = awaitingInitialization
	}
	return nil
}

func (c *MossSocketClient) Run() error {
	if err := c.connect(); err != nil {
		return err
	}
	if err := c.sendInitialization(); err != nil {
		return err
	}
	if err := c.sendLanguage(); err != nil {
		return err
	}
	return nil
}

func (c *MossSocketClient) sendCommand(objects ...interface{}) error {
	commandStrings := make([]string, 0, len(objects))

	for var5 := 0; var5 < len(objects); var5++ {
		o := objects[var5]
		s := fmt.Sprintf("%v", o)
		//s := o.(string)
		commandStrings = append(commandStrings, s)
	}
	if err := c.sendCommandStrings(commandStrings); err != nil {
		return err
	}
	return nil
}

func (c *MossSocketClient) sendCommandStrings(stringSlice []string) error {
	if len(stringSlice) > 0 {
		//slice转字符串,空格分隔
		s := strings.Join(stringSlice, " ")
		s += "\n"
		if _, err := c.conn.Write([]byte(s)); err != nil {
			return errors.New("failed to send command: " + err.Error())
		}
		return nil
	} else {
		return errors.New("failed to send command because it was empty")
	}
}

func (c *MossSocketClient) sendInitialization() error {
	if c.currentStage != awaitingInitialization {
		return errors.New("cannot send initialization. Client is either already initialized or not connected yet")
	}
	if err := c.sendCommand("moss", c.userID); err != nil {
		return nil
	}
	if err := c.sendCommand("directory", c.optD); err != nil {
		return nil
	}
	if err := c.sendCommand("X", c.optX); err != nil {
		return nil
	}
	if err := c.sendCommand("maxmatches", c.optM); err != nil {
		return nil
	}
	if err := c.sendCommand("show", c.optN); err != nil {
		return nil
	}
	c.currentStage = awaitingLanguage
	return nil
}

func (c *MossSocketClient) sendLanguage() error {
	if c.currentStage != awaitingLanguage {
		return errors.New("language already sent or client is not initialized yet")
	}
	if err := c.sendCommand("language", c.language); err != nil {
		return err
	}
	buf := make([]byte, 1024)
	n, err := c.conn.Read(buf)
	if err != nil {
		return err
	}
	receiveString := string(buf[:n-1])
	if receiveString == "yes" {
		c.currentStage = awaitingFiles
	} else {
		return errors.New("MOSS Server does not recognize this programming language")
	}
	return nil
}

func (c *MossSocketClient) sendLanguageWithLanguage(language string) error {
	c.language = language
	if err := c.sendLanguage(); err != nil {
		return err
	}
	return nil
}

func (c *MossSocketClient) SendQuery() error {
	if c.currentStage != awaitingQuery {
		return errors.New("cannot send query at this time. Connection is either not initialized or already closed")
	} else if c.setID == 1 {
		return errors.New("you did not upload any files yet")
	} else {
		if err := c.sendCommand(fmt.Sprintf("%s %d %s", "query", 0, c.optC)); err != nil {
			return nil
		}
		c.currentStage = awaitingResults
		buf := make([]byte, 1024)
		n, err := c.conn.Read(buf)
		if err != nil {
			return err
		}
		receiveString := string(buf[:n-1])
		if strings.HasPrefix(strings.ToLower(receiveString), "http") {
			if c.ResultURL, err = url.Parse(strings.Trim(receiveString, " ")); err != nil {
				return err
			}
			c.currentStage = awaitingEnd
		} else {
			return errors.New("MOSS submission failed. The server did not return a valid URL with detection results")
		}
	}
	return nil
}

func (c *MossSocketClient) UploadFile(filePath string, isBaseFile bool) error {
	if c.currentStage != awaitingFiles && c.currentStage != awaitingQuery {
		return errors.New("cannot upload file. Client is either not initialized properly or the connection is already closed")
	}
	fileBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}
	var setID int
	if isBaseFile {
		setID = 0
	} else {
		setID = c.setID
		c.setID++
	}
	filename := strings.ReplaceAll(filePath, "\\", "/")
	uploadString := fmt.Sprintf("file %d %s %d %s\n", setID, c.language, len(fileBytes), filename)
	println("uploading file: " + filename)
	if _, err = c.conn.Write([]byte(uploadString)); err != nil {
		return err
	}
	if _, err = c.conn.Write(fileBytes); err != nil {
		return err
	}
	c.currentStage = awaitingQuery
	return nil
}
