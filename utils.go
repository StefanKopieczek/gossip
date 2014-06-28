package gossip

import "bytes"

func joinStrings(separator string, elements ...string) (string) {
    var buffer bytes.Buffer

    size := len(elements)
    for idx, element := range(elements) {
        buffer.WriteString(element)

        if (idx < size - 1) {
            buffer.WriteString(separator)
        }
    }

    return buffer.String()
}

func strPtrEq(a *string, b *string) (bool) {
    if a == nil && b == nil {
        return true
    }

    if a == nil || b == nil {
        return false
    }

    return *a == *b
}

func uint16PtrEq(a *uint16, b *uint16) (bool) {
    if a == nil && b == nil {
        return true
    }

    if a == nil || b == nil {
        return false
    }

    return *a == *b
}
