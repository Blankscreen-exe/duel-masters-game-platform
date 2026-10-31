interface ButtonProps {
  disabled?: boolean;
  disabledTooltip?: string;
  onClick?: () => void;
  children: React.ReactNode;
  variant?: "default" | "destructive" | "outline" | "gray";
  autoFocus?: boolean;
  /** "large" gives a finger a bigger target. "narrow" lets the label wrap,
   * for a button in a slim column. */
  size?: "default" | "large" | "narrow";
}

/** The site's button looks. */
const VARIANT_CLASSES = {
  default: "bz-button-primary",
  gray: "bz-button-secondary",
  outline: "bz-button-secondary",
  destructive: "bz-button-danger",
} as const;

const SIZE_CLASSES = {
  default: "px-4 py-2 text-xs whitespace-nowrap",
  large: "px-4 py-3 text-sm whitespace-nowrap",
  narrow: "px-1 py-2 text-[11px] leading-tight",
} as const;

export function Button({
  disabled = false,
  disabledTooltip,
  onClick,
  children,
  variant = "default",
  autoFocus,
  size = "default",
}: ButtonProps) {
  return (
    <div className="group/button relative">
      <button
        disabled={disabled}
        onClick={onClick}
        autoFocus={autoFocus}
        className={`bz-button w-full overflow-hidden text-ellipsis ${SIZE_CLASSES[size]} ${VARIANT_CLASSES[variant]} ${
          disabled ? "" : "cursor-pointer active:scale-95"
        }`}
      >
        {children}
      </button>
      {disabled && disabledTooltip && (
        <div className="absolute bottom-full left-1/2 -translate-x-1/2 mb-2 px-3 py-2 bg-panel text-slate-200 ring-1 ring-slate-500/40 text-xs whitespace-nowrap opacity-0 group-hover/button:opacity-100 transition-opacity pointer-events-none">
          {disabledTooltip}
        </div>
      )}
    </div>
  );
}
