import { Shortcut, ShortcutProvider } from '@slimsag/react-shortcuts'
import React from 'react'

import { KeyboardShortcutWithCallback } from '../constants'

export const ShortcutController: React.FC<{
    shortcuts: KeyboardShortcutWithCallback[]
}> = React.memo(({ shortcuts }) => (
    <ShortcutProvider>
        {shortcuts.map(({ keybindings, onMatch, id }) =>
            keybindings.map((keybinding, index) => (
                <Shortcut key={`${id}-${index}`} {...keybinding} onMatch={onMatch} />
            ))
        )}
    </ShortcutProvider>
))
