import { Link, Route, Routes } from "react-router-dom";
import { ProtectedRoute } from "./auth/AuthContext";
import { AppShell } from "./layout/AppShell";
import DatenschutzPage from "./pages/DatenschutzPage";
import HomePage from "./pages/HomePage";
import ImpressumPage from "./pages/ImpressumPage";
import InvoicePage from "./pages/customer/InvoicePage";
import StatusPage from "./pages/customer/StatusPage";
import DashboardPage from "./pages/workshop/DashboardPage";
import LoginPage from "./pages/workshop/LoginPage";
import OrderDetailPage from "./pages/workshop/OrderDetailPage";
import OrdersPage from "./pages/workshop/OrdersPage";

function NotFoundPage() {
  return (
    <section className="page page--reading">
      <header className="page__header">
        <h1>Seite nicht gefunden</h1>
        <p className="page__subtitle">Diese Adresse gibt es nicht.</p>
      </header>
      <div className="card">
        <div className="card__body">
          <Link to="/">Zur Startseite</Link>
        </div>
      </div>
    </section>
  );
}

/** The complete route table of the contract. */
export default function App() {
  return (
    <Routes>
      <Route element={<AppShell />}>
        <Route path="/" element={<HomePage />} />
        <Route path="/status" element={<StatusPage />} />
        <Route path="/rechnung" element={<InvoicePage />} />
        <Route path="/impressum" element={<ImpressumPage />} />
        <Route path="/datenschutz" element={<DatenschutzPage />} />
        <Route path="/werkstatt/login" element={<LoginPage />} />
        <Route element={<ProtectedRoute />}>
          <Route path="/werkstatt/auftraege" element={<OrdersPage />} />
          <Route
            path="/werkstatt/auftraege/:orderNumber"
            element={<OrderDetailPage />}
          />
          <Route path="/werkstatt/dashboard" element={<DashboardPage />} />
        </Route>
        <Route path="*" element={<NotFoundPage />} />
      </Route>
    </Routes>
  );
}
