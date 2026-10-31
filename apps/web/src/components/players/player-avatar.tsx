interface PlayerAvatarProps {
  readonly username: string;
  /** The picture they chose; without one, the first letter of their name. */
  readonly src: string | undefined;
  readonly size: 'small' | 'large';
}

const SIZES = {
  small: 'cut-corners-sm size-8 text-base',
  large: 'cut-corners size-20 text-4xl',
};

/** A player's picture, beside their name, so it says nothing more to a screen reader. */
export function PlayerAvatar({ username, src, size }: PlayerAvatarProps) {
  const frame = `${SIZES[size]} shrink-0`;

  if (src) {
    return (
      // eslint-disable-next-line @next/next/no-img-element -- tiny pixel art, served ready to use
      <img src={src} alt="" className={`${frame} object-cover [image-rendering:pixelated]`} />
    );
  }

  return (
    <span
      aria-hidden
      className={`${frame} grid place-items-center bg-linear-to-br from-slate-200 to-slate-500 font-display font-bold text-slate-900 transition group-hover:from-white group-hover:to-slate-400`}
    >
      {username.charAt(0).toUpperCase()}
    </span>
  );
}
