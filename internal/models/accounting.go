package models

import (
	"fmt"
	"time"

	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
	"layeh.com/radius/rfc2866"
)

// ======================= ENUM =======================

type AccRecordType int

const (
	Start AccRecordType = iota + 1 // RADIUS uses 1,2,3
	Stop
	Interim
)

// ======================= INTERFACE =======================
type AccountingEvent interface {
	Validate() error
	GenerateRedisKey() string
	GetType() AccRecordType
}

// ======================= BASE STRUCT =====================
type BaseAccountingRecord struct {
	// The username from the RADIUS request (User-Name attribute)
	Username         string `json:"username"`
	// The IP address of the Network Access Server (NAS-IP-Address attribute)
	NASIPAddress     string `json:"nas_ip_address"`
	// The port number on the NAS (NAS-Port attribute)
	NASPort          int    `json:"nas_port"`
	// Unique identifier for the session (Acct-Session-Id attribute)
	AcctSessionID    string `json:"acct_session_id"`
	// The caller's identifier (Calling-Station-Id attribute)
	CallingStationID string `json:"calling_station_id"`
	// The called party's identifier (Called-Station-Id attribute)
	CalledStationID  string `json:"called_station_id"`
	// The IP address of the client making the request
	ClientIP         string `json:"client_ip"`
	// When the accounting request was received
	Timestamp        string `json:"timestamp"`
}

// ======================= SPECIFIC TYPES ==================
type StartRecord struct {
	BaseAccountingRecord
	// IP address assigned to the user (Framed-IP-Address attribute)
	FramedIPAddress string `json:"framed_ip_address"`
}

type StopRecord struct {
	BaseAccountingRecord
	SessionTime    int    `json:"session_time"`
	// This attribute indicates how many seconds the user has received service for.
	TerminateCause string `json:"terminate_cause"`
	// This attribute indicates how many octets have been received from the port over the course of this service being provided.
	InputOctets    uint64 `json:"input_octets"`
	// This attribute indicates how many octets have been sent to the port in the course of delivering this service.
	OutputOctets   uint64 `json:"output_octets"`
}

type InterimRecord struct {
	BaseAccountingRecord
	// This attribute indicates how many seconds the user has received service for.
	SessionTime  int    `json:"session_time"`
	// This attribute indicates how many octets have been received from the port over the course of this service being provided.
	InputOctets  uint64 `json:"input_octets"`
	// This attribute indicates how many octets have been sent to the port in the course of delivering this service.
	OutputOctets uint64 `json:"output_octets"`
}

// ======================= VALIDATION ======================
func (b *BaseAccountingRecord) validateBase() error {
	if b.Username == "" {
		return fmt.Errorf("username is required")
	}
	if b.AcctSessionID == "" {
		return fmt.Errorf("acct session id is required")
	}
	if b.NASIPAddress == "" {
		return fmt.Errorf("NAS IP address is required")
	}
	if b.ClientIP == "" {
		return fmt.Errorf("client IP is required")
	}
	return nil
}

func (r *StartRecord) Validate() error {
	if err := r.validateBase(); err != nil {
		return err
	}
	if r.FramedIPAddress == "" {
		return fmt.Errorf("framed IP address required for Start record")
	}
	return nil
}

func (r *StopRecord) Validate() error {
	if err := r.validateBase(); err != nil {
		return err
	}
	if r.SessionTime == 0 {
		return fmt.Errorf("session time required for Stop record")
	}
	if r.TerminateCause == "" {
		return fmt.Errorf("terminate cause required for Stop record")
	}
	return nil
}

func (r *InterimRecord) Validate() error {
	if err := r.validateBase(); err != nil {
		return err
	}
	if r.SessionTime == 0 {
		return fmt.Errorf("session time required for Interim-Update record")
	}
	return nil
}

// ======================= GET TYPE =========================
func (r *StartRecord) GetType() AccRecordType   { return Start }
func (r *StopRecord) GetType() AccRecordType    { return Stop }
func (r *InterimRecord) GetType() AccRecordType { return Interim }

// ======================= REDIS KEY =========================
func (r *BaseAccountingRecord) keyPrefix() string {
	return fmt.Sprintf("radius:acct:%s:%s:%s", r.Username, r.AcctSessionID, r.Timestamp)
}
func (r *StartRecord) GenerateRedisKey() string   { return "start:" + r.keyPrefix() }
func (r *StopRecord) GenerateRedisKey() string    { return "stop:" + r.keyPrefix() }
func (r *InterimRecord) GenerateRedisKey() string { return "interim:" + r.keyPrefix() }

// ======================= PARSER ===========================
func ParseRADIUSPacket(packet *radius.Packet, clientIP string) (AccountingEvent, error) {
	if packet == nil {
		return nil, fmt.Errorf("packet cannot be nil")
	}

	statusType := rfc2866.AcctStatusType_Get(packet)
	base := BaseAccountingRecord{
		Username:         rfc2865.UserName_GetString(packet),
		NASIPAddress:     rfc2865.NASIPAddress_Get(packet).String(),
		NASPort:          int(rfc2865.NASPort_Get(packet)),
		AcctSessionID:    rfc2866.AcctSessionID_GetString(packet),
		CallingStationID: rfc2865.CallingStationID_GetString(packet),
		CalledStationID:  rfc2865.CalledStationID_GetString(packet),
		ClientIP:         clientIP,
		Timestamp:        time.Now().UTC().Format(time.RFC3339Nano),
	}

	switch statusType {
	case rfc2866.AcctStatusType_Value_Start:
		return &StartRecord{
			BaseAccountingRecord: base,
			FramedIPAddress:      rfc2865.FramedIPAddress_Get(packet).String(),
		}, nil

	case rfc2866.AcctStatusType_Value_Stop:
		return &StopRecord{
			BaseAccountingRecord: base,
			SessionTime:          int(rfc2866.AcctSessionTime_Get(packet)),
			TerminateCause:       fmt.Sprintf("%d", rfc2866.AcctTerminateCause_Get(packet)),
			InputOctets:          uint64(rfc2866.AcctInputOctets_Get(packet)),
			OutputOctets:         uint64(rfc2866.AcctOutputOctets_Get(packet)),
		}, nil

	case rfc2866.AcctStatusType_Value_InterimUpdate:
		return &InterimRecord{
			BaseAccountingRecord: base,
			SessionTime:          int(rfc2866.AcctSessionTime_Get(packet)),
			InputOctets:          uint64(rfc2866.AcctInputOctets_Get(packet)),
			OutputOctets:         uint64(rfc2866.AcctOutputOctets_Get(packet)),
		}, nil

	default:
		return nil, fmt.Errorf("unsupported accounting status type: %d", statusType)
	}
}
