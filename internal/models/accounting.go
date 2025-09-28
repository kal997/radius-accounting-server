package models

import (
	"fmt"
	"time"

	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
	"layeh.com/radius/rfc2866"
)

type AccRecordType int

const (
	Start AccRecordType = iota + 1 // RADIUS uses 1,2,3
	Stop
	Interim
)

type AccountingRecord struct {
	// The username from the RADIUS request (User-Name attribute)
	Username string `json:"username"`

	// The IP address of the Network Access Server (NAS-IP-Address attribute)
	NASIPAddress string `json:"nas_ip_address"`

	// The port number on the NAS (NAS-Port attribute)
	NASPort int `json:"nas_port"`

	// The type of accounting record (Start, Stop, etc.) (Acct-Status-Type attribute)
	AcctStatusType AccRecordType `json:"acct_status_type"`

	// Unique identifier for the session (Acct-Session-Id attribute)
	AcctSessionID string `json:"acct_session_id"`

	// IP address assigned to the user (Framed-IP-Address attribute)
	FramedIPAddress string `json:"framed_ip_address"`

	// The caller's identifier (Calling-Station-Id attribute)
	CallingStationID string `json:"calling_station_id"`

	// The called party's identifier (Called-Station-Id attribute)
	CalledStationID string `json:"called_station_id"`

	// When the accounting request was received
	Timestamp string `json:"timestamp"`

	// The IP address of the client making the request
	ClientIP string `json:"client_ip"`

	// The type of RADIUS packet (e.g., "Accounting-Request", "Accounting-Response")
	PacketType string `json:"packet_type"`
}

// Constructor function
func NewAccountingRecordFromRADIUS(packet *radius.Packet, clientIP string) (*AccountingRecord, error) {
	if packet == nil {
		return nil, fmt.Errorf("packet cannot be nil")
	}

	// Extract attributes from RADIUS packet
	username := rfc2865.UserName_GetString(packet)
	acctStatusType := rfc2866.AcctStatusType_Get(packet)
	acctSessionID := rfc2866.AcctSessionID_GetString(packet)
	nasIPAddress := rfc2865.NASIPAddress_Get(packet)
	nasPort := rfc2865.NASPort_Get(packet)
	framedIPAddress := rfc2865.FramedIPAddress_Get(packet)
	callingStationID := rfc2865.CallingStationID_GetString(packet)
	calledStationID := rfc2865.CalledStationID_GetString(packet)

	// Convert status type to our enum
	var statusType AccRecordType
	switch acctStatusType {
	case rfc2866.AcctStatusType_Value_Start:
		statusType = Start
	case rfc2866.AcctStatusType_Value_Stop:
		statusType = Stop
	case rfc2866.AcctStatusType_Value_InterimUpdate:
		statusType = Interim
	default:
		return nil, fmt.Errorf("unsupported accounting status type: %d", acctStatusType)
	}

	// Create timestamp
	timestamp := time.Now().UTC().Format(time.RFC3339Nano)

	record := &AccountingRecord{
		Username:         username,
		NASIPAddress:     nasIPAddress.String(),
		NASPort:          int(nasPort),
		AcctStatusType:   statusType,
		AcctSessionID:    acctSessionID,
		FramedIPAddress:  framedIPAddress.String(),
		CallingStationID: callingStationID,
		CalledStationID:  calledStationID,
		Timestamp:        timestamp,
		ClientIP:         clientIP,
		PacketType:       "Accounting-Request",
	}

	return record, nil
}

// Validate checks if the accounting record has all required fields
func (ar *AccountingRecord) Validate() error {
	if ar.Username == "" {
		return fmt.Errorf("username is required")
	}
	if ar.AcctSessionID == "" {
		return fmt.Errorf("accounting session ID is required")
	}
	if ar.NASIPAddress == "" {
		return fmt.Errorf("NAS IP address is required")
	}
	if ar.AcctStatusType < Start || ar.AcctStatusType > Interim {
		return fmt.Errorf("invalid accounting status type: %d", ar.AcctStatusType)
	}
	if ar.Timestamp == "" {
		return fmt.Errorf("timestamp is required")
	}
	if ar.ClientIP == "" {
		return fmt.Errorf("client IP is required")
	}
	return nil
}

// GenerateRedisKey creates the unique key for Redis storage
func (ar *AccountingRecord) GenerateRedisKey() string {
	// Format: radius:acct:{username}:{acct_session_id}:{timestamp}
	return fmt.Sprintf("radius:acct:%s:%s:%s", ar.Username, ar.AcctSessionID, ar.Timestamp)
}
