package repoMongo

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/bsonrw"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"reflect"
)

func NewClient(url, username, password, dbName string) (*mongo.Client, error) {
	opts := options.Client().SetRegistry(newMongoUUIDRegistry()).ApplyURI(url)
	if username != "" && password != "" {
		opts.SetAuth(options.Credential{
			Username: username, Password: password, AuthSource: dbName,
		})
	}

	ctx := context.Background()

	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, err
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, err
	}

	return client, nil
}

var (
	tUUID       = reflect.TypeOf(uuid.UUID{})
	uuidSubtype = byte(0x04)
)

func newMongoUUIDRegistry() *bsoncodec.Registry {
	mongoUUIDRegistry := bson.NewRegistry()
	mongoUUIDRegistry.RegisterTypeEncoder(tUUID, bsoncodec.ValueEncoderFunc(uuidEncodeValue))
	mongoUUIDRegistry.RegisterTypeDecoder(tUUID, bsoncodec.ValueDecoderFunc(uuidDecodeValue))

	return mongoUUIDRegistry
}

func uuidEncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tUUID {
		return bsoncodec.ValueEncoderError{Name: "uuidEncodeValue", Types: []reflect.Type{tUUID}, Received: val}
	}
	b := val.Interface().(uuid.UUID)
	return vw.WriteBinaryWithSubtype(b[:], uuidSubtype)
}

func uuidDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tUUID {
		return bsoncodec.ValueDecoderError{Name: "uuidDecodeValue", Types: []reflect.Type{tUUID}, Received: val}
	}

	var data []byte
	var subtype byte
	var err error
	switch vrType := vr.Type(); vrType {
	case 0x05:
		data, subtype, err = vr.ReadBinary()
		if subtype != uuidSubtype {
			return fmt.Errorf("unsupported binary subtype %v for UUID", subtype)
		}
	case 0x0A:
		err = vr.ReadNull()
	case 0x06:
		err = vr.ReadUndefined()
	default:
		return fmt.Errorf("cannot decode %v into a UUID", vrType)
	}

	if err != nil {
		return err
	}
	uuid2, err := uuid.FromBytes(data)
	if err != nil {
		return err
	}
	val.Set(reflect.ValueOf(uuid2))
	return nil
}

// https://gist.github.com/SupaHam/3afe982dc75039356723600ccc91ff77 -- откуда взял
