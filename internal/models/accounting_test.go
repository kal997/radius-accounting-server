package models

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
	"layeh.com/radius/rfc2866"
)

func TestNewAccountingRecordFromRADIUS(t *testing.T) {
	// Valid packet test
	packet := radius.New(radius.CodeAccountingRequest, []byte("secret"))
	rfc2865.UserName_SetString(packet, "testuser")
	rfc2866.AcctStatusType_Set(packet, rfc2866.AcctStatusType_Value_Start)
	rfc2866.AcctSessionID_SetString(packet, "session123")
	rfc2865.NASIPAddress_Set(packet, net.ParseIP("192.168.1.1"))

	record, err := NewAccountingRecordFromRADIUS(packet, "192.168.1.100")

	require.NoError(t, err)
	assert.Equal(t, "testuser", record.Username)
	assert.Equal(t, Start, record.AcctStatusType)
	assert.Equal(t, "session123", record.AcctSessionID)
	assert.NotEmpty(t, record.Timestamp)

	// Nil packet test
	record, err = NewAccountingRecordFromRADIUS(nil, "192.168.1.100")
	assert.Error(t, err)
	assert.Nil(t, record)
}

func TestAccountingRecord_Validate(t *testing.T) {
	// Valid record
	record := &AccountingRecord{
		Username:       "testuser",
		AcctSessionID:  "session123",
		NASIPAddress:   "192.168.1.1",
		AcctStatusType: Start,
		Timestamp:      "2024-01-15T10:30:45Z",
		ClientIP:       "192.168.1.100",
	}
	assert.NoError(t, record.Validate())

	// Invalid record
	record.Username = ""
	assert.Error(t, record.Validate())
}

func TestAccountingRecord_GenerateRedisKey(t *testing.T) {
	record := &AccountingRecord{
		Username:      "testuser",
		AcctSessionID: "session123",
		Timestamp:     "2024-01-15T10:30:45Z",
	}

	key := record.GenerateRedisKey()
	expected := "radius:acct:testuser:session123:2024-01-15T10:30:45Z"
	
	assert.Equal(t, expected, key)
}