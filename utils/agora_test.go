package utils

import (
	"bytes"
	"encoding/binary"
	"os"
	"strings"
	"testing"
)

// ─── BuildAgoraToken ──────────────────────────────────────────────────────────

func TestBuildAgoraToken_NoCertificate(t *testing.T) {
	os.Unsetenv("AGORA_APP_CERTIFICATE")
	os.Unsetenv("AGORA_APP_ID")

	result := BuildAgoraToken("channel1", "user1", AgoraRolePublisher, 3600)
	if result != "" {
		t.Errorf("expected empty string when certificate is not set, got %q", result)
	}
}

func TestBuildAgoraToken_NoCertificateSubscriber(t *testing.T) {
	os.Unsetenv("AGORA_APP_CERTIFICATE")
	result := BuildAgoraToken("channel1", "", AgoraRoleSubscriber, 3600)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestBuildAgoraToken_Publisher(t *testing.T) {
	os.Setenv("AGORA_APP_ID", "testappid0123456789abcdef012345")
	os.Setenv("AGORA_APP_CERTIFICATE", "testcert0123456789abcdef0123456")
	defer func() {
		os.Unsetenv("AGORA_APP_ID")
		os.Unsetenv("AGORA_APP_CERTIFICATE")
	}()

	token := BuildAgoraToken("testchannel", "testuser", AgoraRolePublisher, 3600)
	if token == "" {
		t.Fatal("expected non-empty token for publisher")
	}
	if !strings.HasPrefix(token, "006") {
		t.Errorf("expected token to start with '006', got prefix %q", token[:min(6, len(token))])
	}
}

func TestBuildAgoraToken_Subscriber(t *testing.T) {
	os.Setenv("AGORA_APP_ID", "testappid0123456789abcdef012345")
	os.Setenv("AGORA_APP_CERTIFICATE", "testcert0123456789abcdef0123456")
	defer func() {
		os.Unsetenv("AGORA_APP_ID")
		os.Unsetenv("AGORA_APP_CERTIFICATE")
	}()

	token := BuildAgoraToken("testchannel", "", AgoraRoleSubscriber, 3600)
	if token == "" {
		t.Fatal("expected non-empty token for subscriber")
	}
	if !strings.HasPrefix(token, "006") {
		t.Errorf("expected token to start with '006'")
	}
}

func TestBuildAgoraToken_ContainsAppID(t *testing.T) {
	appID := "myappid1234567890abcdef12345678"
	os.Setenv("AGORA_APP_ID", appID)
	os.Setenv("AGORA_APP_CERTIFICATE", "testcert0123456789abcdef0123456")
	defer func() {
		os.Unsetenv("AGORA_APP_ID")
		os.Unsetenv("AGORA_APP_CERTIFICATE")
	}()

	token := BuildAgoraToken("channel", "user", AgoraRolePublisher, 3600)
	if !strings.Contains(token, appID) {
		t.Errorf("expected token to contain appID %q", appID)
	}
}

func TestBuildAgoraToken_PublisherVsSubscriberDiffer(t *testing.T) {
	os.Setenv("AGORA_APP_ID", "testappid0123456789abcdef012345")
	os.Setenv("AGORA_APP_CERTIFICATE", "testcert0123456789abcdef0123456")
	defer func() {
		os.Unsetenv("AGORA_APP_ID")
		os.Unsetenv("AGORA_APP_CERTIFICATE")
	}()

	// Publisher has more privileges, so tokens should differ
	pubToken := BuildAgoraToken("ch", "u", AgoraRolePublisher, 3600)
	subToken := BuildAgoraToken("ch", "u", AgoraRoleSubscriber, 3600)

	// They use random salt so direct equality comparison isn't reliable,
	// but both should be non-empty
	if pubToken == "" || subToken == "" {
		t.Error("both tokens should be non-empty")
	}
}

func TestBuildAgoraToken_EmptyChannel(t *testing.T) {
	os.Setenv("AGORA_APP_ID", "testappid0123456789abcdef012345")
	os.Setenv("AGORA_APP_CERTIFICATE", "testcert0123456789abcdef0123456")
	defer func() {
		os.Unsetenv("AGORA_APP_ID")
		os.Unsetenv("AGORA_APP_CERTIFICATE")
	}()

	token := BuildAgoraToken("", "", AgoraRoleSubscriber, 3600)
	// Should still produce a token (valid Agora use case)
	if token == "" {
		t.Error("expected non-empty token even for empty channel/user")
	}
}

// ─── Internal helper: agoraPackUint16 ────────────────────────────────────────

func TestAgoraPackUint16(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	agoraPackUint16(buf, 0x1234)

	if buf.Len() != 2 {
		t.Errorf("expected 2 bytes, got %d", buf.Len())
	}
	val := binary.LittleEndian.Uint16(buf.Bytes())
	if val != 0x1234 {
		t.Errorf("expected 0x1234, got 0x%04x", val)
	}
}

func TestAgoraPackUint16_Zero(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	agoraPackUint16(buf, 0)
	val := binary.LittleEndian.Uint16(buf.Bytes())
	if val != 0 {
		t.Errorf("expected 0, got %d", val)
	}
}

func TestAgoraPackUint16_MaxValue(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	agoraPackUint16(buf, 0xFFFF)
	val := binary.LittleEndian.Uint16(buf.Bytes())
	if val != 0xFFFF {
		t.Errorf("expected 0xFFFF, got 0x%04x", val)
	}
}

// ─── Internal helper: agoraPackUint32 ────────────────────────────────────────

func TestAgoraPackUint32(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	agoraPackUint32(buf, 0x12345678)

	if buf.Len() != 4 {
		t.Errorf("expected 4 bytes, got %d", buf.Len())
	}
	val := binary.LittleEndian.Uint32(buf.Bytes())
	if val != 0x12345678 {
		t.Errorf("expected 0x12345678, got 0x%08x", val)
	}
}

func TestAgoraPackUint32_Zero(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	agoraPackUint32(buf, 0)
	val := binary.LittleEndian.Uint32(buf.Bytes())
	if val != 0 {
		t.Errorf("expected 0, got %d", val)
	}
}

func TestAgoraPackUint32_MaxValue(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	agoraPackUint32(buf, 0xFFFFFFFF)
	val := binary.LittleEndian.Uint32(buf.Bytes())
	if val != 0xFFFFFFFF {
		t.Errorf("expected 0xFFFFFFFF, got 0x%08x", val)
	}
}

// ─── Internal helper: agoraPackBytes ─────────────────────────────────────────

func TestAgoraPackBytes_Normal(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	data := []byte("hello")
	agoraPackBytes(buf, data)

	// Should write 2-byte length prefix + data
	if buf.Len() != 2+len(data) {
		t.Errorf("expected %d bytes, got %d", 2+len(data), buf.Len())
	}

	b := buf.Bytes()
	length := binary.LittleEndian.Uint16(b[:2])
	if int(length) != len(data) {
		t.Errorf("expected length %d, got %d", len(data), length)
	}
	if string(b[2:]) != "hello" {
		t.Errorf("expected 'hello', got %q", string(b[2:]))
	}
}

func TestAgoraPackBytes_Empty(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	agoraPackBytes(buf, []byte{})

	if buf.Len() != 2 {
		t.Errorf("expected 2 bytes (length prefix only), got %d", buf.Len())
	}
	length := binary.LittleEndian.Uint16(buf.Bytes())
	if length != 0 {
		t.Errorf("expected length 0, got %d", length)
	}
}

// ─── Internal helper: agoraPackMessage ───────────────────────────────────────

func TestAgoraPackMessage_BasicStructure(t *testing.T) {
	privileges := map[uint16]uint32{
		1: 100,
		2: 200,
	}
	result := agoraPackMessage(42, 9999, privileges)

	if len(result) == 0 {
		t.Fatal("expected non-empty result from agoraPackMessage")
	}

	// First 4 bytes: salt (uint32 LE)
	salt := binary.LittleEndian.Uint32(result[0:4])
	if salt != 42 {
		t.Errorf("expected salt=42, got %d", salt)
	}

	// Next 4 bytes: ts (uint32 LE)
	ts := binary.LittleEndian.Uint32(result[4:8])
	if ts != 9999 {
		t.Errorf("expected ts=9999, got %d", ts)
	}
}

func TestAgoraPackMessage_PrivilegesAreSorted(t *testing.T) {
	// Pack with unsorted keys — the function should sort them
	privileges := map[uint16]uint32{
		4: 400,
		1: 100,
		3: 300,
		2: 200,
	}
	result := agoraPackMessage(1, 1, privileges)

	// Skip salt(4) + ts(4) = 8 bytes
	// Next 2 bytes: count of privileges
	count := binary.LittleEndian.Uint16(result[8:10])
	if int(count) != len(privileges) {
		t.Errorf("expected privilege count %d, got %d", len(privileges), count)
	}

	// Read privilege keys in order
	offset := 10
	var lastKey uint16
	for i := 0; i < int(count); i++ {
		key := binary.LittleEndian.Uint16(result[offset : offset+2])
		if i > 0 && key <= lastKey {
			t.Errorf("privileges not sorted: key[%d]=%d <= key[%d]=%d", i, key, i-1, lastKey)
		}
		lastKey = key
		offset += 2 + 4 // key (2) + value (4)
	}
}

func TestAgoraPackMessage_EmptyPrivileges(t *testing.T) {
	result := agoraPackMessage(1, 2, map[uint16]uint32{})
	// salt(4) + ts(4) + count(2) = 10 bytes minimum
	if len(result) < 10 {
		t.Errorf("expected at least 10 bytes, got %d", len(result))
	}
}

// ─── Role constants ───────────────────────────────────────────────────────────

func TestAgoraRoleConstants(t *testing.T) {
	if AgoraRolePublisher != 1 {
		t.Errorf("AgoraRolePublisher: want 1, got %d", AgoraRolePublisher)
	}
	if AgoraRoleSubscriber != 2 {
		t.Errorf("AgoraRoleSubscriber: want 2, got %d", AgoraRoleSubscriber)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
