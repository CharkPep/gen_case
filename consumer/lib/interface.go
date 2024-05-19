package consumer

import (
	"context"
	"fmt"
	"github.com/dranikpg/gtrs"
	"github.com/redis/go-redis/v9"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

type BankRateMessage struct {
	Bank       string    `gtrs:"bank,required"`
	Buy        float64   `gtrs:"buy"`
	BuyOnline  float64   `gtrs:"buy_online"`
	Sell       float64   `gtrs:"sell"`
	SellOnline float64   `gtrs:"sell_online"`
	UpdateAt   time.Time `gtrs:"update_at,required"`
	SiteUrl    string    `gtrs:"site_url"`
	SourceUrl  string    `gtrs:"source_url,required"`
}

func (b *BankRateMessage) Unmarshal(v map[string]interface{}) error {
	resultValue := reflect.ValueOf(b).Elem()
	resultType := reflect.TypeOf(b).Elem()
	for i := 0; i < resultType.NumField(); i += 1 {
		fieldValue := resultValue.Field(i)
		fieldType := resultType.Field(i)
		fieldKey := getFieldNameFromType(fieldType)
		StrVal, ok := v[fieldKey]
		if !ok {
			continue
		}

		_, ok = StrVal.(string)
		if !ok {
			if isFieldRequired(fieldType) {
				return gtrs.ParseError{
					Data: v,
					Err:  fmt.Errorf("missing required field %s", fieldType.Name),
				}
			}
			continue
		}

		if StrVal == "" {
			if isFieldRequired(fieldType) {
				return gtrs.ParseError{
					Data: v,
					Err:  fmt.Errorf("missing required field %s", fieldType.Name),
				}
			}
			continue
		}

		rVal, err := fieldFromString(fieldValue, StrVal.(string))
		if err != nil {
			return gtrs.ParseError{
				Data: v,
				Err:  err,
			}
		}
		if rVal == nil {
			if isFieldRequired(fieldType) {
				return gtrs.ParseError{
					Data: v,
					Err:  fmt.Errorf("missing required field %s", fieldType.Name),
				}
			}
			continue
		}

		fieldValue.Set(reflect.ValueOf(rVal))
	}

	return nil
}

func fieldFromString(field reflect.Value, str string) (any, error) {
	var (
		rVal any
		err  error
	)
	switch field.Interface().(type) {
	case float64:
		rVal, err = strconv.ParseFloat(str, 64)
		if err != nil {
			return nil, err
		}
	case string:
		rVal = str
	case time.Time:
		rVal, err = time.Parse(time.RFC3339, str)
		rVal.(time.Time).Round(time.Millisecond)
		if err != nil {
			return nil, err
		}
	}

	return rVal, nil
}

func isFieldRequired(fieldType reflect.StructField) bool {
	t := fieldType.Tag
	return slices.ContainsFunc(strings.Split(t.Get("gtrs"), ","), func(e string) bool {
		if e == "required" {
			return true
		}

		return false
	})

}

func toSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

func getFieldNameFromType(fieldType reflect.StructField) string {
	t := fieldType.Tag
	nameItem := strings.SplitN(strings.TrimSpace(t.Get("gtrs")), ",", 2)[0]
	var fieldName string
	if len(nameItem) > 0 {
		fieldName = nameItem
	} else {
		fieldName = toSnakeCase(fieldType.Name)
	}
	return fieldName
}

func CheckAndCreateGroup(ctx context.Context, rdb *redis.Client, stream, group, start string) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	_, err := rdb.XGroupCreateMkStream(ctx, stream, group, start).Result()
	if err != nil {
		// fine for now dont have time))
		if regexp.MustCompile("BUSYGROUP").Match([]byte(err.Error())) {
			return nil
		}
		return err
	}

	return nil
}

func (b *BankRateMessage) FromMap(v map[string]any) error {
	if err := b.Unmarshal(v); err != nil {
		return err
	}

	return nil
}
