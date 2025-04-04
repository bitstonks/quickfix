// Copyright (c) quickfixengine.org  All rights reserved.
//
// This file may be distributed under the terms of the quickfixengine.org
// license as defined by quickfixengine.org and appearing in the file
// LICENSE included in the packaging of this file.
//
// This file is provided AS IS with NO WARRANTY OF ANY KIND, INCLUDING
// THE WARRANTY OF DESIGN, MERCHANTABILITY AND FITNESS FOR A
// PARTICULAR PURPOSE.
//
// See http://www.quickfixengine.org/LICENSE for licensing information.
//
// Contact ask@quickfixengine.org if any conditions of this licensing
// are not clear to you.

package quickfix

import (
	"bytes"
	"io"
	"regexp"
	"time"
)

func writeLoop(connection io.Writer, messageOut chan []byte, log Log) {
	for {
		msg, ok := <-messageOut
		if !ok {
			return
		}

		if _, err := connection.Write(msg); err != nil {
			log.OnEvent(err.Error())
		}
	}
}

func batchWriteLoop(connection io.Writer, messageOut chan []byte, log Log, maxBatchDuration time.Duration, maxBatchSize int) {
	bufferReadyToSend := maxBatchSize * 9 / 10 // 90% of max size
	buffer := bytes.NewBuffer(make([]byte, 0, maxBatchSize))
	tick := time.NewTicker(maxBatchDuration)
	var channelClosed bool

	for {
		tick.Reset(maxBatchDuration)
		buffer.Reset()

	innerLoop:
		for {
			select {
			case msg, ok := <-messageOut:
				if !ok {
					return
				}

				str := string(msg)
				_ = str
				if _, err := buffer.Write(msg); err != nil {
					log.OnEvent(err.Error())
				}

				if buffer.Len() >= bufferReadyToSend || containsAdminMessageTypeBytes(msg) {
					break innerLoop
				}

			case <-tick.C:
				break innerLoop
			}
		}

		if buffer.Len() > 0 {
			if _, err := io.Copy(connection, buffer); err != nil {
				log.OnEvent(err.Error())
			}
		}

		if channelClosed {
			return
		}
	}
}

// adminRE checks for messageType Heartbeat (0), Logon (A), TestRequest (1), ResendRequest (2), Reject (3), SequenceReset (4), Logout (5)
var adminRE = regexp.MustCompile("\x0135=[0A1-5]\x01")

func containsAdminMessageTypeBytes(msg []byte) bool {
	return adminRE.Match(msg)
}

func readLoop(parser *parser, msgIn chan fixIn, log Log) {
	defer close(msgIn)

	for {
		msg, err := parser.ReadMessage()
		if err != nil {
			log.OnEvent(err.Error())
			return
		}
		msgIn <- fixIn{msg, parser.lastRead}
	}
}
