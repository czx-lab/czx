package xlog

import "testing"

func TestLog(t *testing.T) {
	t.Run("LogTest", func(t *testing.T) {
		Write().Info("test log info.")
		Write().Warn("test log warn.")
		Write().Error("test log error.")
		Write().Debug("test log debug.")

		Load(&XLogConf{
			ServiceName: "test",
			Mode:        FileMode,
			Encoding:    EncodingJson,
			Level:       "info",
		})

		Write().Error("test file log error.")
	})
}
