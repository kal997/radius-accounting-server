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

func TestNewAccountingRecordFromRADIUS(t *testing.T) {
	tests := []struct {
		name        string
		setupPacket func() *radius.Packet
		clientIP    string
		wantErr     bool
		errContains string
		validate    func(*testing.T, *AccountingRecord)
	}{
		{
			name: "valid packet with Start status",
			setupPacket: func() *radius.Packet {
				packet := radius.New(radius.CodeAccountingRequest, []byte("secret"))
				rfc2865.UserName_SetString(packet, "testuser")
				rfc2866.AcctStatusType_Set(packet, rfc2866.AcctStatusType_Value_Start)
				rfc2866.AcctSessionID_SetString(packet, "session123")
				rfc2865.NASIPAddress_Set(packet, net.ParseIP("192.168.1.1"))
				rfc2865.NASPort_Set(packet, 1234)
				rfc2865.FramedIPAddress_Set(packet, net.ParseIP("10.0.0.5"))
				rfc2865.CallingStationID_SetString(packet, "00:11:22:33:44:55")
				rfc2865.CalledStationID_SetString(packet, "AP-001")
				return packet
			},
			clientIP: "192.168.1.100",
			wantErr:  false,
			validate: func(t *testing.T, r *AccountingRecord) {
				assert.Equal(t, "testuser", r.Username)
				assert.Equal(t, Start, r.AcctStatusType)
				assert.Equal(t, "session123", r.AcctSessionID)
				assert.Equal(t, "192.168.1.1", r.NASIPAddress)
				assert.Equal(t, 1234, r.NASPort)
				assert.Equal(t, "10.0.0.5", r.FramedIPAddress)
				assert.Equal(t, "00:11:22:33:44:55", r.CallingStationID)
				assert.Equal(t, "AP-001", r.CalledStationID)
				assert.Equal(t, "192.168.1.100", r.ClientIP)
				assert.Equal(t, "Accounting-Request", r.PacketType)
				assert.NotEmpty(t, r.Timestamp)

				// Verify timestamp format
				_, err := time.Parse(time.RFC3339Nano, r.Timestamp)
				assert.NoError(t, err)
			},
		},
		{
			name: "valid packet with Stop status",
			setupPacket: func() *radius.Packet {
				packet := radius.New(radius.CodeAccountingRequest, []byte("secret"))
				rfc2865.UserName_SetString(packet, "user2")
				rfc2866.AcctStatusType_Set(packet, rfc2866.AcctStatusType_Value_Stop)
				rfc2866.AcctSessionID_SetString(packet, "session456")
				rfc2865.NASIPAddress_Set(packet, net.ParseIP("192.168.2.1"))
				return packet
			},
			clientIP: "192.168.2.100",
			wantErr:  false,
			validate: func(t *testing.T, r *AccountingRecord) {
				assert.Equal(t, "user2", r.Username)
				assert.Equal(t, Stop, r.AcctStatusType)
				assert.Equal(t, "session456", r.AcctSessionID)
				assert.Equal(t, "192.168.2.1", r.NASIPAddress)
			},
		},
		{
			name: "valid packet with Interim status",
			setupPacket: func() *radius.Packet {
				packet := radius.New(radius.CodeAccountingRequest, []byte("secret"))
				rfc2865.UserName_SetString(packet, "user3")
				rfc2866.AcctStatusType_Set(packet, rfc2866.AcctStatusType_Value_InterimUpdate)
				rfc2866.AcctSessionID_SetString(packet, "session789")
				rfc2865.NASIPAddress_Set(packet, net.ParseIP("192.168.3.1"))
				return packet
			},
			clientIP: "192.168.3.100",
			wantErr:  false,
			validate: func(t *testing.T, r *AccountingRecord) {
				assert.Equal(t, "user3", r.Username)
				assert.Equal(t, Interim, r.AcctStatusType)
				assert.Equal(t, "session789", r.AcctSessionID)
				assert.Equal(t, "192.168.3.1", r.NASIPAddress)
			},
		},
		{
			name:        "nil packet",
			setupPacket: func() *radius.Packet { return nil },
			clientIP:    "192.168.1.100",
			wantErr:     true,
			errContains: "packet cannot be nil",
		},
		{
			name: "unsupported accounting status type",
			setupPacket: func() *radius.Packet {
				packet := radius.New(radius.CodeAccountingRequest, []byte("secret"))
				rfc2865.UserName_SetString(packet, "testuser")
				// Set an unsupported status type (e.g., 7 for Accounting-On)
				rfc2866.AcctStatusType_Set(packet, 7)
				rfc2866.AcctSessionID_SetString(packet, "session123")
				rfc2865.NASIPAddress_Set(packet, net.ParseIP("192.168.1.1"))
				return packet
			},
			clientIP:    "192.168.1.100",
			wantErr:     true,
			errContains: "unsupported accounting status type: 7",
		},
		{
			name: "packet with nil IP addresses",
			setupPacket: func() *radius.Packet {
				packet := radius.New(radius.CodeAccountingRequest, []byte("secret"))
				rfc2865.UserName_SetString(packet, "testuser")
				rfc2866.AcctStatusType_Set(packet, rfc2866.AcctStatusType_Value_Start)
				rfc2866.AcctSessionID_SetString(packet, "session123")
				// NAS IP and Framed IP will be nil/unset
				return packet
			},
			clientIP: "192.168.1.100",
			wantErr:  false,
			validate: func(t *testing.T, r *AccountingRecord) {
				assert.Equal(t, "testuser", r.Username)
				assert.Equal(t, "<nil>", r.NASIPAddress) // nil IP converts to "<nil>"
				assert.Equal(t, "<nil>", r.FramedIPAddress)
			},
		},
		{
			name: "packet with minimal required fields",
			setupPacket: func() *radius.Packet {
				packet := radius.New(radius.CodeAccountingRequest, []byte("secret"))
				rfc2866.AcctStatusType_Set(packet, rfc2866.AcctStatusType_Value_Start)
				return packet
			},
			clientIP: "192.168.1.100",
			wantErr:  false,
			validate: func(t *testing.T, r *AccountingRecord) {
				assert.Equal(t, "", r.Username) // empty when not set
				assert.Equal(t, "", r.AcctSessionID)
				assert.Equal(t, 0, r.NASPort)
				assert.Equal(t, "", r.CallingStationID)
				assert.Equal(t, "", r.CalledStationID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packet := tt.setupPacket()
			record, err := NewAccountingRecordFromRADIUS(packet, tt.clientIP)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, record)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, record)
				if tt.validate != nil {
					tt.validate(t, record)
				}
			}
		})
	}
}

func TestAccountingRecord_Validate(t *testing.T) {
	tests := []struct {
		name        string
		record      *AccountingRecord
		wantErr     bool
		errContains string
	}{
		{
			name: "valid record",
			record: &AccountingRecord{
				Username:       "testuser",
				AcctSessionID:  "session123",
				NASIPAddress:   "192.168.1.1",
				AcctStatusType: Start,
				Timestamp:      "2024-01-15T10:30:45Z",
				ClientIP:       "192.168.1.100",
			},
			wantErr: false,
		},
		{
			name: "missing username",
			record: &AccountingRecord{
				Username:       "",
				AcctSessionID:  "session123",
				NASIPAddress:   "192.168.1.1",
				AcctStatusType: Start,
				Timestamp:      "2024-01-15T10:30:45Z",
				ClientIP:       "192.168.1.100",
			},
			wantErr:     true,
			errContains: "username is required",
		},
		{
			name: "missing acct session ID",
			record: &AccountingRecord{
				Username:       "testuser",
				AcctSessionID:  "",
				NASIPAddress:   "192.168.1.1",
				AcctStatusType: Start,
				Timestamp:      "2024-01-15T10:30:45Z",
				ClientIP:       "192.168.1.100",
			},
			wantErr:     true,
			errContains: "accounting session ID is required",
		},
		{
			name: "missing NAS IP address",
			record: &AccountingRecord{
				Username:       "testuser",
				AcctSessionID:  "session123",
				NASIPAddress:   "",
				AcctStatusType: Start,
				Timestamp:      "2024-01-15T10:30:45Z",
				ClientIP:       "192.168.1.100",
			},
			wantErr:     true,
			errContains: "NAS IP address is required",
		},
		{
			name: "invalid acct status type - too low",
			record: &AccountingRecord{
				Username:       "testuser",
				AcctSessionID:  "session123",
				NASIPAddress:   "192.168.1.1",
				AcctStatusType: 0, // Less than Start (1)
				Timestamp:      "2024-01-15T10:30:45Z",
				ClientIP:       "192.168.1.100",
			},
			wantErr:     true,
			errContains: "invalid accounting status type: 0",
		},
		{
			name: "invalid acct status type - too high",
			record: &AccountingRecord{
				Username:       "testuser",
				AcctSessionID:  "session123",
				NASIPAddress:   "192.168.1.1",
				AcctStatusType: 4, // Greater than Interim (3)
				Timestamp:      "2024-01-15T10:30:45Z",
				ClientIP:       "192.168.1.100",
			},
			wantErr:     true,
			errContains: "invalid accounting status type: 4",
		},
		{
			name: "missing timestamp",
			record: &AccountingRecord{
				Username:       "testuser",
				AcctSessionID:  "session123",
				NASIPAddress:   "192.168.1.1",
				AcctStatusType: Start,
				Timestamp:      "",
				ClientIP:       "192.168.1.100",
			},
			wantErr:     true,
			errContains: "timestamp is required",
		},
		{
			name: "missing client IP",
			record: &AccountingRecord{
				Username:       "testuser",
				AcctSessionID:  "session123",
				NASIPAddress:   "192.168.1.1",
				AcctStatusType: Start,
				Timestamp:      "2024-01-15T10:30:45Z",
				ClientIP:       "",
			},
			wantErr:     true,
			errContains: "client IP is required",
		},
		{
			name: "valid record with Stop status",
			record: &AccountingRecord{
				Username:       "testuser",
				AcctSessionID:  "session123",
				NASIPAddress:   "192.168.1.1",
				AcctStatusType: Stop,
				Timestamp:      "2024-01-15T10:30:45Z",
				ClientIP:       "192.168.1.100",
			},
			wantErr: false,
		},
		{
			name: "valid record with Interim status",
			record: &AccountingRecord{
				Username:       "testuser",
				AcctSessionID:  "session123",
				NASIPAddress:   "192.168.1.1",
				AcctStatusType: Interim,
				Timestamp:      "2024-01-15T10:30:45Z",
				ClientIP:       "192.168.1.100",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.record.Validate()

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAccountingRecord_GenerateRedisKey(t *testing.T) {
	tests := []struct {
		name     string
		record   *AccountingRecord
		expected string
	}{
		{
			name: "standard key generation",
			record: &AccountingRecord{
				Username:      "testuser",
				AcctSessionID: "session123",
				Timestamp:     "2024-01-15T10:30:45Z",
			},
			expected: "radius:acct:testuser:session123:2024-01-15T10:30:45Z",
		},
		{
			name: "key with special characters",
			record: &AccountingRecord{
				Username:      "user@example.com",
				AcctSessionID: "sess-456-abc",
				Timestamp:     "2024-01-15T10:30:45.123Z",
			},
			expected: "radius:acct:user@example.com:sess-456-abc:2024-01-15T10:30:45.123Z",
		},
		{
			name: "key with empty fields",
			record: &AccountingRecord{
				Username:      "",
				AcctSessionID: "",
				Timestamp:     "",
			},
			expected: "radius:acct:::",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := tt.record.GenerateRedisKey()
			assert.Equal(t, tt.expected, key)
		})
	}
}

func TestAccRecordType_Constants(t *testing.T) {
	// Verify the constant values match RADIUS standard
	assert.Equal(t, AccRecordType(1), Start)
	assert.Equal(t, AccRecordType(2), Stop)
	assert.Equal(t, AccRecordType(3), Interim)
}
