import { FitAddon } from "@xterm/addon-fit";
import { Terminal } from "@xterm/xterm";
import { useEffect, useRef, useState } from "react";

type ConnectionStatus = "connecting" | "connected" | "disconnected";

const reconnectDelay = 1_000;
const touchScrollSensitivity = 2;
const textEncoder = new TextEncoder();

type ToolbarKey = {
  label: string;
  value: string;
  ariaLabel?: string;
};

const toolbarKeys: ToolbarKey[] = [
  { label: "Esc", value: "\x1b" },
  { label: "Tab", value: "\t" },
  { label: "C-b", value: "\x02", ariaLabel: "Control B, tmux prefix" },
  {
    label: "C-f",
    value: "\x06",
    ariaLabel: "Control F, switch tmux session",
  },
  { label: "←", value: "\x1b[D", ariaLabel: "Left arrow" },
  { label: "↓", value: "\x1b[B", ariaLabel: "Down arrow" },
  { label: "↑", value: "\x1b[A", ariaLabel: "Up arrow" },
  { label: "→", value: "\x1b[C", ariaLabel: "Right arrow" },
];

function websocketURL(session?: string) {
  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  const query = session ? `?session=${encodeURIComponent(session)}` : "";
  return `${protocol}//${window.location.host}/ws${query}`;
}

export function TerminalView({ session }: { session: string }) {
  const containerRef = useRef<HTMLDivElement>(null);
  const sendInputRef = useRef<(data: string) => void>(() => undefined);
  const ctrlArmedRef = useRef(false);
  const [status, setStatus] = useState<ConnectionStatus>("connecting");
  const [ctrlArmed, setCtrlArmed] = useState(false);

  const setCtrl = (armed: boolean) => {
    ctrlArmedRef.current = armed;
    setCtrlArmed(armed);
  };

  const sendToolbarKey = (data: string) => {
    setCtrl(false);
    sendInputRef.current(data);
  };

  useEffect(() => {
    const container = containerRef.current;
    if (!container) {
      return;
    }

    const terminal = new Terminal({
      cursorBlink: true,
      fontFamily: '"JetBrainsMono Nerd Font", monospace',
      fontSize: 14,
      theme: {
        background: "#0b0b0b",
        cursor: "#e68e0d",
        foreground: "#e7e2d9",
        selectionBackground: "#e68e0d55",
      },
    });
    const fitAddon = new FitAddon();
    terminal.loadAddon(fitAddon);
    terminal.open(container);

    let socket: WebSocket | null = null;
    let reconnectTimer: number | undefined;
    let fitFrame = 0;
    let lastTouchY: number | undefined;
    let touchX = 0;
    let touchY = 0;
    let touchDeltaY = 0;
    let touchScrollFrame = 0;
    let hasConnected = false;
    let disposed = false;

    const sendInput = (data: string) => {
      if (socket?.readyState === WebSocket.OPEN) {
        socket.send(textEncoder.encode(data));
      }
    };
    sendInputRef.current = sendInput;

    const sendResize = () => {
      if (socket?.readyState !== WebSocket.OPEN) {
        return;
      }
      socket.send(
        JSON.stringify({
          type: "resize",
          cols: terminal.cols,
          rows: terminal.rows,
        }),
      );
    };

    const fit = () => {
      if (container.clientWidth === 0 || container.clientHeight === 0) {
        return;
      }
      fitAddon.fit();
      sendResize();
    };

    const scheduleFit = () => {
      window.cancelAnimationFrame(fitFrame);
      fitFrame = window.requestAnimationFrame(fit);
    };

    const updateViewport = () => {
      const height = window.visualViewport?.height ?? window.innerHeight;
      document.documentElement.style.setProperty(
        "--viewport-height",
        `${height}px`,
      );
      scheduleFit();
    };

    const connect = () => {
      if (
        disposed ||
        socket?.readyState === WebSocket.OPEN ||
        socket?.readyState === WebSocket.CONNECTING
      ) {
        return;
      }

      window.clearTimeout(reconnectTimer);
      setStatus("connecting");
      const nextSocket = new WebSocket(websocketURL(hasConnected ? undefined : session));
      socket = nextSocket;
      nextSocket.binaryType = "arraybuffer";

      nextSocket.addEventListener("open", () => {
        if (disposed) {
          return;
        }
        hasConnected = true;
        setStatus("connected");
        scheduleFit();
        terminal.focus();
      });

      nextSocket.addEventListener("message", (event) => {
        terminal.write(
          typeof event.data === "string"
            ? event.data
            : new Uint8Array(event.data as ArrayBuffer),
        );
      });

      nextSocket.addEventListener("close", () => {
        if (disposed || socket !== nextSocket) {
          return;
        }
        socket = null;
        setStatus("disconnected");
        if (document.visibilityState === "visible") {
          reconnectTimer = window.setTimeout(connect, reconnectDelay);
        }
      });
    };

    const input = terminal.onData((data) => {
      let nextData = data;
      if (ctrlArmedRef.current) {
        setCtrl(false);
        const firstCharacter = data[0];
        if (firstCharacter && /^[a-z]$/i.test(firstCharacter)) {
          nextData =
            String.fromCharCode(firstCharacter.toUpperCase().charCodeAt(0) - 64) +
            data.slice(1);
        }
      }
      sendInput(nextData);
    });
    const handleTouchStart = (event: TouchEvent) => {
      lastTouchY = event.touches.length === 1 ? event.touches[0].clientY : undefined;
    };
    const flushTouchScroll = () => {
      touchScrollFrame = 0;
      if (touchDeltaY === 0 || !terminal.element) {
        return;
      }

      const wheelTarget =
        terminal.element.querySelector<HTMLElement>(".xterm-scrollable-element") ??
        terminal.element;
      wheelTarget.dispatchEvent(
        new WheelEvent("wheel", {
          bubbles: true,
          cancelable: true,
          clientX: touchX,
          clientY: touchY,
          deltaMode: WheelEvent.DOM_DELTA_PIXEL,
          deltaY: touchDeltaY * touchScrollSensitivity,
          view: window,
        }),
      );
      touchDeltaY = 0;
    };
    const handleTouchMove = (event: TouchEvent) => {
      if (lastTouchY === undefined || event.touches.length !== 1 || !terminal.element) {
        return;
      }

      const touch = event.touches[0];
      const deltaY = lastTouchY - touch.clientY;
      lastTouchY = touch.clientY;
      touchX = touch.clientX;
      touchY = touch.clientY;
      if (deltaY === 0) {
        return;
      }

      touchDeltaY += deltaY;
      if (touchScrollFrame === 0) {
        touchScrollFrame = window.requestAnimationFrame(flushTouchScroll);
      }
      event.preventDefault();
    };
    const handleTouchEnd = () => {
      lastTouchY = undefined;
    };
    const resizeObserver = new ResizeObserver(scheduleFit);
    const handleVisibilityChange = () => {
      if (document.visibilityState === "visible") {
        connect();
        updateViewport();
      } else {
        window.clearTimeout(reconnectTimer);
      }
    };

    resizeObserver.observe(container);
    window.addEventListener("resize", updateViewport);
    window.visualViewport?.addEventListener("resize", updateViewport);
    document.addEventListener("visibilitychange", handleVisibilityChange);
    window.addEventListener("online", connect);
    container.addEventListener("touchstart", handleTouchStart, { passive: true });
    container.addEventListener("touchmove", handleTouchMove, { passive: false });
    container.addEventListener("touchend", handleTouchEnd);
    container.addEventListener("touchcancel", handleTouchEnd);
    updateViewport();
    connect();

    return () => {
      disposed = true;
      window.clearTimeout(reconnectTimer);
      window.cancelAnimationFrame(fitFrame);
      window.cancelAnimationFrame(touchScrollFrame);
      resizeObserver.disconnect();
      window.removeEventListener("resize", updateViewport);
      window.visualViewport?.removeEventListener("resize", updateViewport);
      document.removeEventListener("visibilitychange", handleVisibilityChange);
      window.removeEventListener("online", connect);
      container.removeEventListener("touchstart", handleTouchStart);
      container.removeEventListener("touchmove", handleTouchMove);
      container.removeEventListener("touchend", handleTouchEnd);
      container.removeEventListener("touchcancel", handleTouchEnd);
      document.documentElement.style.removeProperty("--viewport-height");
      input.dispose();
      sendInputRef.current = () => undefined;
      socket?.close();
      terminal.dispose();
    };
  }, [session]);

  return (
    <>
      <div className={`connection-status ${status}`} aria-live="polite">
        <span aria-hidden="true" />
        {status}
      </div>
      <div className="terminal" ref={containerRef} aria-label="Pickle terminal" />
      <nav className="terminal-toolbar" aria-label="Terminal keys">
        <button
          className={ctrlArmed ? "active" : undefined}
          type="button"
          aria-pressed={ctrlArmed}
          disabled={status !== "connected"}
          onPointerDown={(event) => event.preventDefault()}
          onClick={() => setCtrl(!ctrlArmed)}
        >
          Ctrl
        </button>
        {toolbarKeys.map((key) => (
          <button
            key={key.label}
            type="button"
            aria-label={key.ariaLabel ?? key.label}
            disabled={status !== "connected"}
            onPointerDown={(event) => event.preventDefault()}
            onClick={() => sendToolbarKey(key.value)}
          >
            {key.label}
          </button>
        ))}
      </nav>
    </>
  );
}
