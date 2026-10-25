interface CardImageProps {
  /** The picture's address, from the API; without one, the card's shape is kept empty. */
  readonly src: string | undefined;
  readonly name: string;
  readonly className?: string;
  /** Load at once, for pictures seen without scrolling. */
  readonly priority?: boolean;
}

/**
 * A card picture at the card's own proportions (491 × 682), so the page does not jump as pictures
 * arrive. The Go server already serves small, cacheable WebP files, so the picture is used as it
 * is rather than through Next.js's image resizing.
 */
export function CardImage({ src, name, className = '', priority = false }: CardImageProps) {
  const frame = `aspect-[491/682] rounded-[4.5%] bg-slate-900 shadow-md ring-1 ring-black/40 ${className}`;

  if (!src) {
    return (
      <div
        role="img"
        aria-label={`${name} (no picture)`}
        className={`${frame} grid place-items-center p-2 text-center font-display text-xs text-slate-500`}
      >
        {name}
      </div>
    );
  }

  return (
    // eslint-disable-next-line @next/next/no-img-element -- served ready to use; see above
    <img
      src={src}
      alt={name}
      loading={priority ? 'eager' : 'lazy'}
      decoding="async"
      className={`${frame} object-cover`}
    />
  );
}
