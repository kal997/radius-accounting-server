package models

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
	"layeh.com/radius/rfc2866"
)

func TestValidate_BaseFields(t *testing.T) {
	tests := []struct {
		name    string
		record  BaseAccountingRecord
		wantErr string
	}{
		{
			name:    "Missing Username",
			record:  BaseAccountingRecord{},
			wantErr: "username is required",
		},
		{
			name: "Missing AcctSessionID",
			record: BaseAccountingRecord{
				Username:     "user",
				NASIPAddress: "1.1.1.1",
				ClientIP:     "2.2.2.2",
			},
			wantErr: "acct session id is required",
		},
		{
			name: "Missing NAS IP",
			record: BaseAccountingRecord{
				Username:      "user",
				AcctSessionID: "sess",
				ClientIP:      "2.2.2.2",
			},
			wantErr: "NAS IP address is required",
		},
		{
			name: "Missing Client IP",
			record: BaseAccountingRecord{
				Username:      "user",
				NASIPAddress:  "1.1.1.1",
				AcctSessionID: "sess",
			},
			wantErr: "client IP is required",
		},
		{
			name: "All Fields Valid",
			record: BaseAccountingRecord{
				Username:      "user",
				NASIPAddress:  "1.1.1.1",
				AcctSessionID: "sess",
				ClientIP:      "2.2.2.2",
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.record.validateBase()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidate_StartRecord(t *testing.T) {
	r := &StartRecord{
		BaseAccountingRecord: BaseAccountingRecord{
			Username:      "testuser",
			NASIPAddress:  "192.168.1.1",
			AcctSessionID: "session123",
			ClientIP:      "192.168.1.100",
			Timestamp:     time.Now().Format(time.RFC3339Nano),
		},
		FramedIPAddress: "",
	}
	err := r.Validate()
	assert.ErrorContains(t, err, "framed IP address required")

	r.FramedIPAddress = "10.0.0.1"
	assert.NoError(t, r.Validate())
}

func TestParseRADIUSPacket_AllTypes(t *testing.T) {
	tests := []struct {
		name       string
		statusType rfc2866.AcctStatusType
		wantType   AccRecordType
	}{
		{"Start", rfc2866.AcctStatusType_Value_Start, Start},
		{"Stop", rfc2866.AcctStatusType_Value_Stop, Stop},
		{"Interim", rfc2866.AcctStatusType_Value_InterimUpdate, Interim},
		{"Unsupported", rfc2866.AcctStatusType(9999), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packet := radius.New(radius.CodeAccountingRequest, []byte("secret"))
			packet.Add(rfc2865.UserName_Type, radius.Attribute("testuser"))

			ipAttr, err := radius.NewIPAddr(net.ParseIP("127.0.0.1"))
			require.NoError(t, err)
			packet.Add(rfc2865.NASIPAddress_Type, ipAttr)

			packet.Add(rfc2866.AcctSessionID_Type, radius.Attribute("session123"))
			rfc2866.AcctStatusType_Set(packet, tt.statusType)

			event, err := ParseRADIUSPacket(packet, "192.168.1.10")

			if tt.wantType == 0 {
				assert.Error(t, err)
				assert.Nil(t, event)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantType, event.GetType())
			}
		})
	}
}

func TestValidate_StopRecord(t *testing.T) {
	r := &StopRecord{
		BaseAccountingRecord: BaseAccountingRecord{
			Username:      "testuser",
			NASIPAddress:  "192.168.1.1",
			AcctSessionID: "session123",
			ClientIP:      "192.168.1.100",
			Timestamp:     time.Now().Format(time.RFC3339Nano),
		},
	}
	err := r.Validate()
	assert.ErrorContains(t, err, "session time required")

	r.SessionTime = 100
	err = r.Validate()
	assert.ErrorContains(t, err, "terminate cause required")

	r.TerminateCause = "User-Request"
	assert.NoError(t, r.Validate())
}

func TestValidate_InterimRecord(t *testing.T) {
	r := &InterimRecord{
		BaseAccountingRecord: BaseAccountingRecord{
			Username:      "testuser",
			NASIPAddress:  "192.168.1.1",
			AcctSessionID: "session123",
			ClientIP:      "192.168.1.100",
			Timestamp:     time.Now().Format(time.RFC3339Nano),
		},
	}
	err := r.Validate()
	assert.ErrorContains(t, err, "session time required")

	r.SessionTime = 500
	assert.NoError(t, r.Validate())
}

func TestGenerateRedisKey(t *testing.T) {
	base := BaseAccountingRecord{
		Username:      "user",
		AcctSessionID: "sess123",
		Timestamp:     "2025-10-04T15:00:00Z",
	}

	start := &StartRecord{BaseAccountingRecord: base}
	stop := &StopRecord{BaseAccountingRecord: base}
	interim := &InterimRecord{BaseAccountingRecord: base}

	assert.Contains(t, start.GenerateRedisKey(), "radius:acct:user:sess123:2025-10-04T15:00:00Z:start")
	assert.Contains(t, stop.GenerateRedisKey(), "radius:acct:user:sess123:2025-10-04T15:00:00Z:stop")
	assert.Contains(t, interim.GenerateRedisKey(), "radius:acct:user:sess123:2025-10-04T15:00:00Z:interim")
}

func TestParseRADIUSPacket_Start(t *testing.T) {
	p := radius.New(radius.CodeAccountingRequest, []byte("secret"))
	rfc2866.AcctStatusType_Set(p, rfc2866.AcctStatusType_Value_Start)
	rfc2865.UserName_SetString(p, "testuser")
	rfc2865.NASIPAddress_Set(p, net.ParseIP("192.168.1.1"))
	rfc2865.FramedIPAddress_Set(p, net.ParseIP("10.0.0.100"))
	rfc2866.AcctSessionID_SetString(p, "sess123")

	event, err := ParseRADIUSPacket(p, "127.0.0.1")
	require.NoError(t, err)

	start, ok := event.(*StartRecord)
	require.True(t, ok)
	assert.Equal(t, "testuser", start.Username)
	assert.Equal(t, "10.0.0.100", start.FramedIPAddress)
	assert.Equal(t, Start, start.GetType())
	assert.NoError(t, start.Validate())
}

func TestParseRADIUSPacket_Stop(t *testing.T) {
	p := radius.New(radius.CodeAccountingRequest, []byte("secret"))
	rfc2866.AcctStatusType_Set(p, rfc2866.AcctStatusType_Value_Stop)
	rfc2865.UserName_SetString(p, "testuser")
	rfc2866.AcctSessionID_SetString(p, "sess456")
	rfc2865.NASIPAddress_Set(p, net.ParseIP("192.168.1.1"))
	rfc2866.AcctSessionTime_Set(p, 600)
	rfc2866.AcctTerminateCause_Set(p, rfc2866.AcctTerminateCause_Value_UserRequest)

	event, err := ParseRADIUSPacket(p, "127.0.0.1")
	require.NoError(t, err)

	stop, ok := event.(*StopRecord)
	require.True(t, ok)
	assert.Equal(t, "testuser", stop.Username)
	assert.Equal(t, Stop, stop.GetType())
	assert.NotEmpty(t, stop.TerminateCause)
	assert.NoError(t, stop.Validate())
}

func TestParseRADIUSPacket_Interim(t *testing.T) {
	p := radius.New(radius.CodeAccountingRequest, []byte("secret"))
	rfc2866.AcctStatusType_Set(p, rfc2866.AcctStatusType_Value_InterimUpdate)
	rfc2865.UserName_SetString(p, "testuser")
	rfc2866.AcctSessionID_SetString(p, "sess789")
	rfc2865.NASIPAddress_Set(p, net.ParseIP("192.168.1.1"))
	rfc2866.AcctSessionTime_Set(p, 900)
	rfc2866.AcctInputOctets_Set(p, 111)
	rfc2866.AcctOutputOctets_Set(p, 222)

	event, err := ParseRADIUSPacket(p, "127.0.0.1")
	require.NoError(t, err)

	interim, ok := event.(*InterimRecord)
	require.True(t, ok)
	assert.Equal(t, "testuser", interim.Username)
	assert.Equal(t, Interim, interim.GetType())
	assert.Equal(t, 900, interim.SessionTime)
	assert.NoError(t, interim.Validate())
}

func TestParseRADIUSPacket_InvalidCases(t *testing.T) {
	// nil packet
	event, err := ParseRADIUSPacket(nil, "1.1.1.1")
	assert.ErrorContains(t, err, "packet cannot be nil")
	assert.Nil(t, event)

	// unsupported status
	p := radius.New(radius.CodeAccountingRequest, []byte("secret"))
	rfc2866.AcctStatusType_Set(p, 99)
	event, err = ParseRADIUSPacket(p, "1.1.1.1")
	assert.ErrorContains(t, err, "unsupported accounting status type: 99")
	assert.Nil(t, event)
}

func TestAccRecordTypeValues(t *testing.T) {
	assert.Equal(t, 1, int(Start))
	assert.Equal(t, 2, int(Stop))
	assert.Equal(t, 3, int(Interim))
}
