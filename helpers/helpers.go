package helpers

import (
    "fmt"
    "strings"
)

func WrapErr(err error, msg string) error {
    if err == nil {
        return nil
    }
    return fmt.Errorf("%s: %w", msg, err)
}

func RemoveFromSliceStringsExistsInOtherSlice(slice1 []string, slice2 []string) []string {
    for i := 0; i < len(slice1); i++ {
        for j := 0; j < len(slice2); j++ {
            if strings.Contains(slice1[i], slice2[j]) {
                slice1 = append(slice1[:i], slice1[i+1:]...)
                i--
                break
            }
        }
    }
    return slice1
}