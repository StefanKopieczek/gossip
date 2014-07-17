package gossip

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
