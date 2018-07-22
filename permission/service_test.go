package permission

import (
	"fmt"
	"testing"
	"time"
)

func initService() *Service {
	p := NewService()

	p.SetToken("t1", "u1", time.Now().Add(time.Hour*2))
	p.SetToken("t2", "u2", time.Now().Add(time.Hour*2))
	p.SetToken("t3", "u3", time.Now().Add(time.Hour*2))
	p.SetToken("t4", "u3", time.Now().Add(time.Hour*-2))
	p.SetUser("u1", []string{"r1", "r2"})
	p.SetUser("u2", []string{"r2"})
	p.SetUser("u3", []string{"r3"})
	p.SetRole("r1", []string{"p1", "p2", "p3"})
	p.SetRole("r2", []string{"p2", "p4"})
	p.SetPermission("p1", "permission1")
	p.SetPermission("p2", "permission2")
	p.SetPermission("p3", "permission3")
	p.SetPermission("p4", "permission4")
	p.SetPermission("p5", "permission5")

	numPermissions := 10000
	r3permissions := make([]string, numPermissions)
	for i := 0; i < numPermissions; i++ {
		p.SetPermission(fmt.Sprintf("p%v", i), fmt.Sprintf("permission%v", i))
		r3permissions[i] = fmt.Sprintf("p%v", i)
	}
	p.SetPermission(fmt.Sprintf("p%v", numPermissions), fmt.Sprintf("permission%v", numPermissions))
	p.SetRole("r3", r3permissions)

	p.BuildUserPermissionData()

	return p
}

func BenchmarkService_PermissionDoesntExist(b *testing.B) {
	p := initService()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.TokenHasPermission("t3", "permission9999999")
	}
}

func BenchmarkService_TokenDoesntExist(b *testing.B) {
	p := initService()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.TokenHasPermission("t99999999", "permission9999999")
	}
}

func BenchmarkService_TokenExpired(b *testing.B) {
	p := initService()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.TokenHasPermission("t4", "permission1")
	}
}

func BenchmarkService_HasPermission(b *testing.B) {
	p := initService()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.TokenHasPermission("t3", "permission1")
	}
}

func BenchmarkService_DoesntHavePermission(b *testing.B) {
	p := initService()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.TokenHasPermission("t1", "permission1")
	}
}

func BenchmarkService_DoesntHavePermission_ManyPermissions(b *testing.B) {
	p := initService()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.TokenHasPermission("t3", "permission10000")
	}
}

func TestService_TokenHasPermission(t *testing.T) {
	p := initService()

	type args struct {
		accessToken string
		permission  string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"permission granted", args{"t1", "permission1"}, true},
		{"permission granted", args{"t1", "permission2"}, true},
		{"permission granted", args{"t1", "permission3"}, true},
		{"permission granted", args{"t1", "permission4"}, true},
		{"permission granted", args{"t3", "permission999"}, true},
		{"permission not granted", args{"t1", "permission5"}, false},
		{"permission doesnt exist", args{"t1", "permission999999"}, false},
		{"permission not granted", args{"t3", "permission10000"}, false},
		{"token expired", args{"t4", "permission1"}, false},
		{"token doesnt exist", args{"t99999", "permission1"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := p.TokenHasPermission(tt.args.accessToken, tt.args.permission); got != tt.want {
				t.Errorf("Service.TokenHasPermission() = %v, want %v", got, tt.want)
			}
		})
	}
}
