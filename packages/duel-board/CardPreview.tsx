import { useEffect } from "react";
import { Button } from "./Button";
import { CardDetails } from "./CardDetails";
import { useCardPictures } from "./cardImage";
import type { StatusBadge } from "./creatureStatus";
import type { CreaturePower } from "./types";

interface CardPreviewProps {
  visible: boolean;
  imageId: string | null;
  name: string | null;
  /** A creature's power, when it is in a battle zone. */
  power?: CreaturePower | null;
  /** A creature's statuses, when it is in a battle zone. */
  statuses?: StatusBadge[];
  onClose: () => void;
}

export function CardPreview({
  visible,
  imageId,
  name,
  power,
  statuses = [],
  onClose,
}: CardPreviewProps) {
  const pictures = useCardPictures();

  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === "Escape" && visible) {
        onClose();
      }
    };

    if (visible) {
      document.addEventListener("keydown", handleEscape);
      document.body.style.overflow = "hidden";
    }

    return () => {
      document.removeEventListener("keydown", handleEscape);
      document.body.style.overflow = "";
    };
  }, [visible, onClose]);

  const handleOverlayClick = (e: React.MouseEvent) => {
    if (e.target === e.currentTarget) {
      onClose();
    }
  };

  if (!visible) return null;

  const imageUrl = pictures.url(imageId);
  // The card gives up some height to the details shown under it, so the
  // close button stays on screen.
  const hasDetails = !!power || statuses.length > 0;

  return (
    <div
      className="bz-scrim fixed inset-0 flex items-center justify-center p-4"
      style={{ zIndex: 1100 }}
      onClick={handleOverlayClick}
    >
      <div className="flex flex-col items-center gap-4">
        <img
          src={imageUrl}
          alt="Card preview"
          className="shadow-2xl"
          style={{
            maxHeight: hasDetails ? "62vh" : "80vh",
            maxWidth: "90vw",
            height: "auto",
            width: "auto",
            borderRadius: "5%",
          }}
        />
        <CardDetails power={power} statuses={statuses} />
        <Button variant="gray" onClick={onClose}>
          Close Preview
        </Button>
      </div>
    </div>
  );
}
