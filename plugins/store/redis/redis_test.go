package redis

import (
	"os"
	"testing"
	"time"

	"github.com/asim/go-micro/v3/store"
	"github.com/go-redis/redis/v8"
)

func Test_rkv_configure(t *testing.T) {
	type fields struct {
		options store.Options
		Client  *redis.Client
	}
	type wantValues struct {
		username string
		password string
		address  string
	}

	tests := []struct {
		name    string
		fields  fields
		wantErr bool
		want    wantValues
	}{
		{name: "No Url", fields: fields{options: store.Options{}, Client: nil},
			wantErr: false, want: wantValues{
			username: "",
			password: "",
			address:  "127.0.0.1:6379",
		}},
		{name: "legacy Url", fields: fields{options: store.Options{Nodes: []string{"127.0.0.1:6379"}}, Client: nil},
			wantErr: false, want: wantValues{
			username: "",
			password: "",
			address:  "127.0.0.1:6379",
		}},
		{name: "New Url", fields: fields{options: store.Options{Nodes: []string{"redis://127.0.0.1:6379"}}, Client: nil},
			wantErr: false, want: wantValues{
			username: "",
			password: "",
			address:  "127.0.0.1:6379",
		}},
		{name: "Url with Pwd", fields: fields{options: store.Options{Nodes: []string{"redis://:password@redis:6379"}}, Client: nil},
			wantErr: false, want: wantValues{
			username: "",
			password: "password",
			address:  "redis:6379",
		}},
		{name: "Url with username and Pwd", fields: fields{options: store.Options{Nodes: []string{"redis://username:password@redis:6379"}}, Client: nil},
			wantErr: false, want: wantValues{
			username: "username",
			password: "password",
			address:  "redis:6379",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &rkv{
				options: tt.fields.options,
				Client:  tt.fields.Client,
			}
			err := r.configure()
			if (err != nil) != tt.wantErr {
				t.Errorf("configure() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if r.Client.Options().Addr != tt.want.address {
				t.Errorf("configure() Address = %v, want address %v", r.Client.Options().Addr, tt.want.address)
			}
			if r.Client.Options().Password != tt.want.password {
				t.Errorf("configure() password = %v, want password %v", r.Client.Options().Password, tt.want.password)
			}
			if r.Client.Options().Username != tt.want.username {
				t.Errorf("configure() username = %v, want username %v", r.Client.Options().Username, tt.want.username)
			}

		})
	}
}

func Test_Store(t *testing.T) {
	if tr := os.Getenv("TRAVIS"); len(tr) > 0 {
		t.Skip()
	}
	r := new(rkv)

	//r.options = store.Options{Nodes: []string{"redis://:password@127.0.0.1:6379"}}
	//r.options = store.Options{Nodes: []string{"127.0.0.1:6379"}}
	r.options = store.Options{Nodes: []string{"redis://127.0.0.1:6379"}}

	if err := r.configure(); err != nil {
		t.Error(err)
		return
	}

	key := "myTest"
	rec := store.Record{
		Key:    key,
		Value:  []byte("myValue"),
		Expiry: 2 * time.Minute,
	}

	err := r.Write(&rec)
	if err != nil {
		t.Errorf("Write Erroe. Error: %v", err)
	}
	rec1, err := r.Read(key)
	if err != nil {
		t.Errorf("Read Error. Error: %v\n", err)
	}
	err = r.Delete(rec1[0].Key)
	if err != nil {
		t.Errorf("Delete error %v\n", err)
	}
	_, err = r.List()
	if err != nil {
		t.Errorf("listing error %v\n", err)
	}
}
