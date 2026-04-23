package utils

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"hash/crc32"
	"math/rand"
	"os"
	"sort"
	"time"
)

const (
	agoraPrivJoinChannel  uint16 = 1
	agoraPrivPublishAudio uint16 = 2
	agoraPrivPublishVideo uint16 = 3
	agoraPrivPublishData  uint16 = 4

	AgoraRolePublisher  = 1
	AgoraRoleSubscriber = 2
)

func agoraPackUint16(buf *bytes.Buffer, n uint16) {
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, n)
	buf.Write(b)
}

func agoraPackUint32(buf *bytes.Buffer, n uint32) {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, n)
	buf.Write(b)
}

func agoraPackBytes(buf *bytes.Buffer, data []byte) {
	agoraPackUint16(buf, uint16(len(data)))
	buf.Write(data)
}

func agoraPackMessage(salt, ts uint32, privileges map[uint16]uint32) []byte {
	buf := bytes.NewBuffer(nil)
	agoraPackUint32(buf, salt)
	agoraPackUint32(buf, ts)

	keys := make([]uint16, 0, len(privileges))
	for k := range privileges {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	agoraPackUint16(buf, uint16(len(keys)))
	for _, k := range keys {
		agoraPackUint16(buf, k)
		agoraPackUint32(buf, privileges[k])
	}
	return buf.Bytes()
}

// BuildAgoraToken generates an Agora RTC V006 token.
// userAccount: "" means any user (UID-based joining with uid=0).
// role: AgoraRolePublisher (teacher) or AgoraRoleSubscriber (student).
// Returns empty string if AGORA_APP_CERTIFICATE is not set (no-token mode).
func BuildAgoraToken(channelName, userAccount string, role int, expireSeconds uint32) string {
	appID := os.Getenv("AGORA_APP_ID")
	appCert := os.Getenv("AGORA_APP_CERTIFICATE")

	if appCert == "" {
		return ""
	}

	ts := uint32(time.Now().Unix()) + expireSeconds
	salt := rand.Uint32()

	privileges := map[uint16]uint32{
		agoraPrivJoinChannel: ts,
	}
	if role == AgoraRolePublisher {
		privileges[agoraPrivPublishAudio] = ts
		privileges[agoraPrivPublishVideo] = ts
		privileges[agoraPrivPublishData] = ts
	}

	val := agoraPackMessage(salt, ts, privileges)

	// Signing string per Agora V006 spec
	toSign := appID + channelName + userAccount + hex.EncodeToString(val)

	h := hmac.New(sha256.New, []byte(appCert))
	h.Write([]byte(toSign))
	signature := h.Sum(nil)

	crcVal := crc32.ChecksumIEEE([]byte(appCert + toSign))

	content := bytes.NewBuffer(nil)
	agoraPackUint32(content, crcVal)
	agoraPackBytes(content, signature)
	content.Write(val)

	return "006" + appID + base64.StdEncoding.EncodeToString(content.Bytes())
}
