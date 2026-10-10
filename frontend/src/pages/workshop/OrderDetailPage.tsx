import { useParams } from "react-router-dom";
import { ComingSoon } from "../../components/ComingSoon";

/**
 * Workshop order detail (route "/werkstatt/auftraege/:orderNumber"). Stub:
 * confirm, item capture and status changes are implemented by their own ticket.
 */
export default function OrderDetailPage() {
  const { orderNumber } = useParams<{ orderNumber: string }>();

  return (
    <section className="page">
      <header className="page__header">
        <h1>Auftrag {orderNumber}</h1>
        <p className="page__subtitle">
          Details, Positionen und Statuswechsel dieses Auftrags.
        </p>
      </header>
      <div className="card">
        <div className="card__body">
          <ComingSoon note="Die Auftragsdetails werden gerade fertiggestellt." />
        </div>
      </div>
    </section>
  );
}
