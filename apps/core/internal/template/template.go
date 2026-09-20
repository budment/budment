package template

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"math/rand/v2"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/budment/budment/internal/fastconv"
	"github.com/dop251/goja"
)

type ScopeProvider interface {
	Resolve(name string) (any, bool)
}

type Expression struct {
	Raw      string
	compiled *FastTemplate
}

func NewExpression(text string) Expression {
	return Expression{
		Raw:      text,
		compiled: Compile(text),
	}
}

func (e Expression) Render(ctx ScopeProvider) string {
	if e.compiled == nil {
		return e.Raw
	}
	return e.compiled.Render(ctx)
}

func (e Expression) IsStatic() bool {
	return e.compiled == nil || (len(e.compiled.chunks) == 1 && e.compiled.chunks[0].Kind == ChunkStatic)
}

type fileEntry struct {
	raw     []byte
	b64     string
	b64Once sync.Once
}

// Loaded only when the worker actively executes.
var lazyFileCache sync.Map

// Helper function to load files with "Thundering Herd" protection (thousands of VUs requesting the file at the same millisecond)
func loadFileEntry(path string) *fileEntry {
	// 1. Fast-path cache lookup
	if val, ok := lazyFileCache.Load(path); ok {
		return val.(*fileEntry)
	}

	// 2. Read from disk if not cached
	data, err := os.ReadFile(path)
	if err != nil {
		data = []byte{} // Prevent panic by returning empty slice if file does not exist
	}

	newEntry := &fileEntry{raw: data}

	// 3. Prevent Race Conditions via LoadOrStore:
	// If 100 VUs access the disk simultaneously, whichever succeeds first stores its entry; redundant copies are discarded.
	actual, _ := lazyFileCache.LoadOrStore(path, newEntry)
	return actual.(*fileEntry)
}

// Retrieves raw byte buffer (Used for open(path, 'b') in JS Hook) -> Avoids RAM overhead from Base64 generation
func GetFileBytes(path string) []byte {
	return bytes.Clone(loadFileEntry(path).raw)
}

func GetFileContent(path string) string {
	return fastconv.BytesToString(loadFileEntry(path).raw)
}

func GetFileBase64(path string) string {
	entry := loadFileEntry(path)

	// Ensures that even if 10,000 VUs call GetFileBase64 concurrently,
	// the encoding algorithm runs exactly once and caches the result in memory.
	entry.b64Once.Do(func() {
		entry.b64 = base64.StdEncoding.EncodeToString(entry.raw)
	})

	return entry.b64
}

type ChunkKind uint8

const (
	ChunkStatic ChunkKind = iota
	ChunkVar
	ChunkEnv
	ChunkFile
	ChunkRandomUUID
	ChunkRandomString
	ChunkRandomInt
	ChunkRandomPick
)

type Chunk struct {
	Kind     ChunkKind
	Value    string
	Raw      string
	IntParam int
	Fallback string
	MaxParam int
	Options  []string
	IsBinary bool
	Path     []string
}

type FastTemplate struct {
	chunks []Chunk
	length int
}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func FastUUID() string {
	var b [16]byte
	u1 := rand.Uint64()
	u2 := rand.Uint64()
	for i := 0; i < 8; i++ {
		b[i] = byte(u1 >> (i * 8))
		b[i+8] = byte(u2 >> (i * 8))
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	var buf [36]byte
	hex.Encode(buf[0:8], b[0:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], b[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], b[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], b[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:36], b[10:])
	return string(buf[:])
}

func FastRandomString(length int) string {
	if length <= 0 {
		return ""
	}
	b := make([]byte, length)
	for i := 0; i < length; i++ {
		b[i] = charset[rand.IntN(len(charset))]
	}
	return fastconv.BytesToString(b)
}

func FastRandomInt(min, max int) int {
	if min > max {
		min, max = max, min
	}
	delta := uint64(max - min + 1)
	if delta == 0 {
		return min
	}
	return min + int(rand.Uint64N(delta))
}

func RandomPick[T any](arr []T) T {
	var zero T
	if len(arr) == 0 {
		return zero
	}
	return arr[rand.IntN(len(arr))]
}

func FastRandomPick(arr []any) any {
	return RandomPick(arr)
}

func GetEnv(key string, fallback ...string) string {
	val := os.Getenv(key)
	if val == "" && len(fallback) > 0 {
		return fallback[0]
	}
	return val
}

func Compile(text string) *FastTemplate {
	if !strings.Contains(text, "{{") {
		return &FastTemplate{
			chunks: []Chunk{{Kind: ChunkStatic, Value: text}},
			length: len(text),
		}
	}

	var chunks []Chunk
	estLen, rem := 0, text

	for {
		start := strings.Index(rem, "{{")
		if start == -1 {
			if len(rem) > 0 {
				chunks = append(chunks, Chunk{Kind: ChunkStatic, Value: rem})
				estLen += len(rem)
			}
			break
		}
		if start > 0 {
			chunks = append(chunks, Chunk{Kind: ChunkStatic, Value: rem[:start]})
			estLen += start
		}
		end := strings.Index(rem[start:], "}}")
		if end == -1 {
			chunks = append(chunks, Chunk{Kind: ChunkStatic, Value: rem[start:]})
			estLen += len(rem[start:])
			break
		}

		rawChunk := rem[start : start+end+2]
		varName := strings.TrimSpace(rem[start+2 : start+end])

		chunk := parseASTChunk(varName, rawChunk)
		estLen += 16
		chunks = append(chunks, chunk)
		rem = rem[start+end+2:]
	}

	return &FastTemplate{chunks: chunks, length: estLen}
}

func parseASTChunk(varName, rawChunk string) Chunk {
	if after, match := strings.CutPrefix(varName, "@env:"); match {
		key, fallback, _ := strings.Cut(after, ":")
		return Chunk{Kind: ChunkEnv, Value: key, Fallback: fallback, Raw: rawChunk}
	}

	if after, match := strings.CutPrefix(varName, "@open:"); match {
		// Stores only the file path in the AST; file contents are not read at this stage.
		filePath := after
		isBinary := false
		if lastColon := strings.LastIndexByte(after, ':'); lastColon != -1 {
			mode := after[lastColon+1:]
			if mode == "b" || mode == "t" {
				filePath = after[:lastColon]
				isBinary = (mode == "b")
			}
		}
		return Chunk{Kind: ChunkFile, Value: filePath, IsBinary: isBinary, Raw: rawChunk}
	}

	if after, match := strings.CutPrefix(varName, "@random:"); match {
		cmd, arg, _ := strings.Cut(after, ":")
		switch cmd {
		case "uuid":
			return Chunk{Kind: ChunkRandomUUID, Raw: rawChunk}
		case "string":
			length := 16
			if arg != "" {
				if l, err := strconv.Atoi(arg); err == nil && l > 0 {
					length = l
				} else {
					return Chunk{Kind: ChunkStatic, Value: rawChunk}
				}
			}
			return Chunk{Kind: ChunkRandomString, IntParam: length, Raw: rawChunk}
		case "int":
			if minStr, maxStr, ok := strings.Cut(arg, ":"); ok {
				minVal, err1 := strconv.Atoi(minStr)
				maxVal, err2 := strconv.Atoi(maxStr)
				if err1 == nil && err2 == nil {
					return Chunk{Kind: ChunkRandomInt, IntParam: minVal, MaxParam: maxVal, Raw: rawChunk}
				}
			}
			return Chunk{Kind: ChunkStatic, Value: rawChunk}
		case "pick":
			if arg != "" {
				options := strings.Split(arg, ",")
				if len(options) > 0 {
					return Chunk{Kind: ChunkRandomPick, Options: options, Raw: rawChunk}
				}
			}
			return Chunk{Kind: ChunkStatic, Value: rawChunk}
		}
	}
	parts := strings.Split(varName, "}{")
	if len(parts) > 1 {
		return Chunk{
			Kind:  ChunkVar,
			Value: parts[0],
			Path:  parts[1:],
			Raw:   rawChunk,
		}
	}

	return Chunk{Kind: ChunkVar, Value: varName, Raw: rawChunk}
}

func (ft *FastTemplate) Render(ctx ScopeProvider) string {
	if len(ft.chunks) == 1 && ft.chunks[0].Kind == ChunkStatic {
		return ft.chunks[0].Value
	}

	var sb strings.Builder
	sb.Grow(ft.length + 32)

	for i := range ft.chunks {
		c := &ft.chunks[i]
		switch c.Kind {
		case ChunkStatic:
			sb.WriteString(c.Value)

		case ChunkEnv:
			sb.WriteString(GetEnv(c.Value, c.Fallback))

		case ChunkFile:
			if c.IsBinary {
				sb.WriteString(GetFileBase64(c.Value))
			} else {
				sb.WriteString(GetFileContent(c.Value))
			}

		case ChunkRandomUUID:
			sb.WriteString(FastUUID())

		case ChunkRandomString:
			sb.WriteString(FastRandomString(c.IntParam))

		case ChunkRandomInt:
			sb.WriteString(strconv.Itoa(FastRandomInt(c.IntParam, c.MaxParam)))

		case ChunkRandomPick:
			if len(c.Options) > 0 {
				sb.WriteString(c.Options[rand.IntN(len(c.Options))])
			}

		case ChunkVar:
			if ctx == nil {
				sb.WriteString(c.Raw)
				continue
			}

			// Get Root Object
			val, exists := ctx.Resolve(c.Value)
			if !exists {
				sb.WriteString(c.Raw)
				continue
			}

			// NO PATH case (e.g., {{user}})
			if len(c.Path) == 0 {
				switch v := val.(type) {
				case *goja.Object:
					jsonBytes, _ := json.Marshal(v.Export())
					sb.Write(jsonBytes)
				case map[string]any, []any:
					jsonBytes, _ := json.Marshal(v)
					sb.Write(jsonBytes)
				default:
					sb.WriteString(fastconv.String(val))
				}
				continue
			}

			// WITH PATH case (e.g., {{user}{items}{0}{id}})
			current := val
			found := true

			for i, p := range c.Path {
				isLast := (i == len(c.Path)-1)

				switch node := current.(type) {

				// Branch A: JavaScript Object
				case *goja.Object:
					propVal := node.Get(p) // propVal returns a goja.Value
					if propVal == nil || goja.IsUndefined(propVal) || goja.IsNull(propVal) {
						found = false
						break
					}

					if isLast {
						current = propVal.Export()
					} else {
						// If intermediate node, attempt type assertion to *goja.Object to continue
						if nextObj, ok := propVal.(*goja.Object); ok {
							current = nextObj
						} else {
							// If JS array or primitive, Export() for handling in branches below
							current = propVal.Export()
						}
					}

				// Branch B: Pure Go Map
				case map[string]any:
					if next, ok := node[p]; ok {
						current = next
						continue
					}
					found = false

				// Branch C: Pure Go Slice/Array (Fix array index traversal e.g., items.0.id)
				case []any:
					idx, err := strconv.Atoi(p)
					if err == nil && idx >= 0 && idx < len(node) {
						current = node[idx]
						continue
					}
					found = false

				default:
					found = false
				}

				if !found {
					break
				}
			}

			// Output final result
			if found && current != nil {
				switch v := current.(type) {
				case map[string]any, []any:
					jsonBytes, _ := json.Marshal(v)
					sb.Write(jsonBytes)
				default:
					sb.WriteString(fastconv.String(current))
				}
			} else {
				sb.WriteString(c.Raw)
			}
		}
	}
	return sb.String()
}
