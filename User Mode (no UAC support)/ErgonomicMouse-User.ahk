#SingleInstance Force

; --- F5/F6/F7 mouse mapping with anti-repeat, micro‑nudge, auto‑elevate,
;     and Scroll Lock master enable/disable (AHK v2) ---

; Fix for multi-monitor jumps: makes the script aware of different monitor scaling
DllCall("SetThreadDpiAwarenessContext", "ptr", -3)

; --- Fast injection and minimal latency ---
SendMode("Input")
SetMouseDelay(-1)
SetDefaultMouseSpeed(0)

; Optional: increase system timer resolution for tighter timing
DllCall("winmm\timeBeginPeriod", "UInt", 1)
OnExit(ExitFunc)
ExitFunc(*) {
    DllCall("winmm\timeEndPeriod", "UInt", 1)
}

; --- Micro-nudge size (pixels). Set to 0 to disable. ---
nudgePixels := 2

NudgeMouseRightLeft() {
    global nudgePixels
    if (nudgePixels <= 0)
        return
    MouseMove(nudgePixels, 0, 0, "R")
    MouseMove(-nudgePixels, 0, 0, "R")
}

; --- Hold-state flags (prevent key auto-repeat while key is held) ---
leftHeld   := false
middleHeld := false
rightHeld  := false

; -----------------------------------------------------------------
;  SCROLL LOCK MASTER TOGGLE  (Enable/Disable all mappings)
;   - Scroll Lock ON  -> mappings enabled
;   - Scroll Lock OFF -> mappings disabled
; -----------------------------------------------------------------

ApplyScrollLockState() {
    ; "T" = toggle state (true if Scroll Lock LED/state is ON)
    state := GetKeyState("ScrollLock","T")
    if state {
        Suspend(0) ; mappings enabled
        ; Optional: TrayTip("Mappings: ON", "", 700)
    } else {
        Suspend(1) ; mappings disabled
        ; Optional: TrayTip("Mappings: OFF", "", 700)
    }
}

; Sync to current Scroll Lock state at startup
ApplyScrollLockState()

; EXEMPT the ScrollLock hotkey so it works even while suspended
#SuspendExempt
~ScrollLock:: {
    Sleep(10)          ; allow OS to update the toggle before we read it
    ApplyScrollLockState()
}
#SuspendExempt False

; -----------------------------------------------------------------
;  MAPPINGS (active only when not Suspended)
; -----------------------------------------------------------------

; F5 = Left button (press=Down, release=Up) with anti-repeat + micro-nudge on first Down
F5:: {
    global leftHeld
    if leftHeld
        return
    Click("D")                 ; left-button down
    leftHeld := true
    NudgeMouseRightLeft()      ; help some apps detect drag/selection immediately
}
F5 Up:: {
    global leftHeld
    if leftHeld {
        Click("U")             ; left-button up
        leftHeld := false
    }
}

; F6 = Middle button (press=Down, release=Up) with anti-repeat (for pan/hand-scroll hold)
F6:: {
    global middleHeld
    if middleHeld
        return
    Click("middle", "D")       ; hold to pan/hand-scroll
    middleHeld := true
}
F6 Up:: {
    global middleHeld
    if middleHeld {
        Click("middle", "U")
        middleHeld := false
    }
}

; F7 = Right button (press=Down, release=Up) with anti-repeat + micro-nudge on first Down
F7:: {
    global rightHeld
    if rightHeld
        return
    Click("right", "D")
    rightHeld := true
    NudgeMouseRightLeft()
}
F7 Up:: {
    global rightHeld
    if rightHeld {
        Click("right", "U")
        rightHeld := false
    }
}

; Optional: panic release if anything ever gets stuck
^F12:: {
    Click("U")
    Click("right", "U")
    Click("middle", "U")
    ; Optional: TrayTip("Released all buttons", "", 600)
}
