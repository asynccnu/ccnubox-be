package viperx

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/spf13/viper"
)

// UnmarshalKeyStrict 读取配置并进行严格校验
func MustUnmarshall(key string, target interface{}) error {
	err := viper.UnmarshalKey(key, target)
	if err != nil {
		return err
	}

	// 校验逻辑
	return validateKeys(key, target)
}

func validateKeys(key string, target interface{}) error {
	val := reflect.ValueOf(target)

	if val.Kind() == reflect.Ptr {
		if val.IsNil() { // 若为 nil 说明 viper 对 target 没有一点动作 即完全没有找到配置模块
			return fmt.Errorf("missing config for section: %s", key)
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		if key != "" && !viper.IsSet(key) {
			return fmt.Errorf("missing config key: '%s'", key)
		}
		return nil
	}

	// 遍历字段递归检查
	t := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := t.Field(i)
		fieldVal := val.Field(i)

		// viper默认是无法写入非导出字段的
		if !field.IsExported() {
			continue
		}

		// 获取标签值
		tag := field.Tag.Get("mapstructure")
		if tag == "" {
			tag = field.Tag.Get("yaml")
		}

		// yaml:"service_name,omitempty" 选取 service_name
		tagName := strings.Split(tag, ",")[0]

		if tagName == "-" {
			continue
		}

		// 若无标签值，猜测它在配置文件里应该叫什么
		if tagName == "" {
			tagName = strings.ToLower(field.Name)
		}

		nextKey := tagName
		if key != "" {
			nextKey = key + "." + tagName
		}

		err := validateKeys(nextKey, fieldVal.Interface())
		if err != nil {
			return err
		}
	}

	return nil
}
