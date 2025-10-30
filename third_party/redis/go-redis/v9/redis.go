package redis

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// Nil represents a nil reply returned by Redis.
var Nil = errors.New("redis: nil")

type Options struct {
	Addr string
}

type Client struct {
	addr string
}

func NewClient(opt *Options) *Client {
	return &Client{addr: opt.Addr}
}

type StatusCmd struct {
	val string
	err error
}

func (c *StatusCmd) Err() error {
	return c.err
}

func (c *StatusCmd) Result() (string, error) {
	return c.val, c.err
}

type IntCmd struct {
	val int64
	err error
}

func (c *IntCmd) Err() error {
	return c.err
}

func (c *IntCmd) Result() (int64, error) {
	return c.val, c.err
}

type StringStringMapCmd struct {
	val map[string]string
	err error
}

func (c *StringStringMapCmd) Err() error {
	return c.err
}

func (c *StringStringMapCmd) Result() (map[string]string, error) {
	return c.val, c.err
}

type StringSliceCmd struct {
	val []string
	err error
}

func (c *StringSliceCmd) Err() error {
	return c.err
}

func (c *StringSliceCmd) Result() ([]string, error) {
	return c.val, c.err
}

func (c *Client) Ping(ctx context.Context) *StatusCmd {
	resp, err := c.do(ctx, []string{"PING"})
	if err != nil {
		return &StatusCmd{err: err}
	}
	str, _ := resp.(string)
	return &StatusCmd{val: str}
}

func (c *Client) HSet(ctx context.Context, key string, values map[string]interface{}) *IntCmd {
	args := []string{"HSET", key}
	for field, value := range values {
		args = append(args, field, fmt.Sprint(value))
	}
	resp, err := c.do(ctx, args)
	if err != nil {
		return &IntCmd{err: err}
	}
	intVal, ok := resp.(int64)
	if !ok {
		return &IntCmd{err: fmt.Errorf("unexpected response type %T", resp)}
	}
	return &IntCmd{val: intVal}
}

func (c *Client) HGetAll(ctx context.Context, key string) *StringStringMapCmd {
	resp, err := c.do(ctx, []string{"HGETALL", key})
	if err != nil {
		return &StringStringMapCmd{err: err}
	}
	arr, ok := resp.([]interface{})
	if !ok {
		return &StringStringMapCmd{err: fmt.Errorf("unexpected response type %T", resp)}
	}
	result := make(map[string]string, len(arr)/2)
	for i := 0; i+1 < len(arr); i += 2 {
		keyStr, _ := arr[i].(string)
		valStr, _ := arr[i+1].(string)
		result[keyStr] = valStr
	}
	return &StringStringMapCmd{val: result}
}

func (c *Client) LPush(ctx context.Context, key string, values ...string) *IntCmd {
	args := []string{"LPUSH", key}
	args = append(args, values...)
	resp, err := c.do(ctx, args)
	if err != nil {
		return &IntCmd{err: err}
	}
	intVal, ok := resp.(int64)
	if !ok {
		return &IntCmd{err: fmt.Errorf("unexpected response type %T", resp)}
	}
	return &IntCmd{val: intVal}
}

func (c *Client) BRPop(ctx context.Context, timeout time.Duration, keys ...string) *StringSliceCmd {
	if len(keys) == 0 {
		return &StringSliceCmd{err: errors.New("no keys provided")}
	}
	seconds := int(timeout / time.Second)
	args := []string{"BRPOP"}
	args = append(args, keys...)
	args = append(args, strconv.Itoa(seconds))

	resp, err := c.do(ctx, args)
	if err != nil {
		return &StringSliceCmd{err: err}
	}
	if resp == nil {
		return &StringSliceCmd{err: Nil}
	}
	arr, ok := resp.([]interface{})
	if !ok {
		return &StringSliceCmd{err: fmt.Errorf("unexpected response type %T", resp)}
	}
	result := make([]string, 0, len(arr))
	for _, item := range arr {
		str, _ := item.(string)
		result = append(result, str)
	}
	return &StringSliceCmd{val: result}
}

func (c *Client) do(ctx context.Context, args []string) (interface{}, error) {
	conn, err := net.Dial("tcp", c.addr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	} else {
		_ = conn.SetDeadline(time.Now().Add(24 * time.Hour))
	}

	if err := writeCommand(conn, args); err != nil {
		return nil, err
	}
	reader := bufio.NewReader(conn)
	return readReply(reader)
}

func writeCommand(conn net.Conn, args []string) error {
	builder := &strings.Builder{}
	builder.Grow(len(args) * 16)
	builder.WriteString("*")
	builder.WriteString(strconv.Itoa(len(args)))
	builder.WriteString("\r\n")
	for _, arg := range args {
		builder.WriteString("$")
		builder.WriteString(strconv.Itoa(len(arg)))
		builder.WriteString("\r\n")
		builder.WriteString(arg)
		builder.WriteString("\r\n")
	}
	_, err := conn.Write([]byte(builder.String()))
	return err
}

func readReply(r *bufio.Reader) (interface{}, error) {
	prefix, err := r.ReadByte()
	if err != nil {
		return nil, err
	}

	switch prefix {
	case '+':
		line, err := readLine(r)
		if err != nil {
			return nil, err
		}
		return line, nil
	case '-':
		line, err := readLine(r)
		if err != nil {
			return nil, err
		}
		return nil, errors.New(line)
	case ':':
		line, err := readLine(r)
		if err != nil {
			return nil, err
		}
		v, err := strconv.ParseInt(line, 10, 64)
		if err != nil {
			return nil, err
		}
		return v, nil
	case '$':
		lengthLine, err := readLine(r)
		if err != nil {
			return nil, err
		}
		length, err := strconv.Atoi(lengthLine)
		if err != nil {
			return nil, err
		}
		if length == -1 {
			return nil, nil
		}
		data := make([]byte, length)
		if _, err := r.Read(data); err != nil {
			return nil, err
		}
		if err := consumeCRLF(r); err != nil {
			return nil, err
		}
		return string(data), nil
	case '*':
		lengthLine, err := readLine(r)
		if err != nil {
			return nil, err
		}
		length, err := strconv.Atoi(lengthLine)
		if err != nil {
			return nil, err
		}
		if length == -1 {
			return nil, nil
		}
		items := make([]interface{}, 0, length)
		for i := 0; i < length; i++ {
			item, err := readReply(r)
			if err != nil {
				return nil, err
			}
			items = append(items, item)
		}
		return items, nil
	default:
		return nil, fmt.Errorf("unexpected prefix byte: %q", prefix)
	}
}

func readLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimSuffix(line, "\n")
	line = strings.TrimSuffix(line, "\r")
	return line, nil
}

func consumeCRLF(r *bufio.Reader) error {
	b1, err := r.ReadByte()
	if err != nil {
		return err
	}
	b2, err := r.ReadByte()
	if err != nil {
		return err
	}
	if b1 != '\r' || b2 != '\n' {
		return errors.New("protocol error: expected CRLF")
	}
	return nil
}
