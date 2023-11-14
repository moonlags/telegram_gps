package types

import "github.com/NicoNex/echotron/v3"

type StateFn func(*echotron.Update) StateFn

type User struct {
	GithubID int64
	IMEI     string
}

type DeviceData struct {
	IMEI                 string
	Posititions          []PosititioningPacket
	BatteryPower         uint8
	LastStatusPacketTime int64
	StatusCooldown       int32
	IsLoggedIn           bool
	InChargingState      bool
}

type PosititioningPacket struct {
	Latitude  float32
	Longitude float32
	Speed     uint8
	Heading   uint16
	Timestamp int64
}
