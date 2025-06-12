package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Appointment struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	PatientID   primitive.ObjectID `bson:"patientId" json:"patientId"`
	PatientName string             `bson:"patientName" json:"patientName"`
	StartTime   time.Time          `bson:"startTime" json:"startTime"`
	EndTime     time.Time          `bson:"endTime" json:"endTime"`
	Service     string             `bson:"service" json:"service"`
	Status      string             `bson:"status" json:"status"`
}
