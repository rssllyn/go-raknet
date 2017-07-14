package extended

import (
	"io"
	"testing"
)

type ConnReadOperation struct {
	ReadBytes    int
	ExpectedData string
}

type ConnReadTestCase struct {
	Packets []string
	Reads   []*ConnReadOperation
}

var (
	ConnReadTestCases = []*ConnReadTestCase{
		{
			[]string{"first", "second"},
			[]*ConnReadOperation{
				{5, "first"},
				{6, "second"},
			},
		},
		{
			[]string{"first", "second"},
			[]*ConnReadOperation{
				{4, "firs"},
				{7, "tsecond"},
			},
		},
		{
			[]string{"first", "second"},
			[]*ConnReadOperation{
				{6, "firsts"},
				{5, "econd"},
			},
		},
		{
			[]string{"first", "", "second"},
			[]*ConnReadOperation{
				{6, "firsts"},
				{5, "econd"},
			},
		},
	}
)

func TestConnRead(t *testing.T) {
	for _, testcase := range ConnReadTestCases {
		conn := &Conn{
			buff:        []byte{},
			buffReadIdx: 0,
			chData:      make(chan []byte, ConnPacketBuffer),
		}
		for _, packet := range testcase.Packets {
			conn.chData <- []byte(packet)
		}
		for _, read := range testcase.Reads {
			buff := make([]byte, read.ReadBytes)
			if _, err := io.ReadFull(conn, buff); err != nil {
				t.Errorf("Conn.Read expects \"%s\", got error", string(read.ExpectedData), err)
				return
			}
			if string(buff) != read.ExpectedData {
				t.Errorf("Conn.Read expects \"%s\", got %s", string(read.ExpectedData), string(buff))
				return
			}
		}
	}
}
