package registry

import (
	"fmt"
	"testing"
	"time"
)

func TestTime(t *testing.T) {

	unix1 := time.Now().Unix()

	time.Sleep(5 * time.Second)

	unix2 := time.Now().Unix()

	fmt.Println(unix2 - unix1)

}
