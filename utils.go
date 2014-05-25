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
