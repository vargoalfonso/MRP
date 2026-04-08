package loghelper

import (
	"context"
	"github.com/sirupsen/logrus"
)

func Infoln(ctx context.Context, args ...interface{}) {
	logrus.WithContext(ctx).Infoln(args...)
}

func Errorln(ctx context.Context, args ...interface{}) {
	logrus.WithContext(ctx).Errorln(args...)
}

func Fatalln(ctx context.Context, args ...interface{}) {
	logrus.WithContext(ctx).Fatalln(args...)
}

func Warningln(ctx context.Context, args ...interface{}) {
	logrus.WithContext(ctx).Warningln(args...)
}
