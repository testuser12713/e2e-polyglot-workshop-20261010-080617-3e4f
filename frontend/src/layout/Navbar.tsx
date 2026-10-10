import { useState } from "react";
import { NavLink, useLocation } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";

interface NavItem {
  to: string;
  label: string;
}

const CUSTOMER_LINKS: NavItem[] = [
  { to: "/", label: "Termin anfragen" },
  { to: "/status", label: "Status verfolgen" },
  { to: "/rechnung", label: "Rechnung" },
];

const WORKSHOP_LINKS: NavItem[] = [
  { to: "/werkstatt/auftraege", label: "Aufträge" },
  { to: "/werkstatt/dashboard", label: "Dashboard" },
];

function linkClass({ isActive }: { isActive: boolean }): string {
  return isActive ? "navbar__link is-active" : "navbar__link";
}

export function Navbar() {
  const location = useLocation();
  const { employee, isAuthenticated, signOut } = useAuth();
  const [open, setOpen] = useState(false);

  const inWorkshop = location.pathname.startsWith("/werkstatt");
  const links = !inWorkshop
    ? CUSTOMER_LINKS
    : isAuthenticated
      ? WORKSHOP_LINKS
      : [];
  const sectionLabel = inWorkshop ? "Werkstattbereich" : "Kundenbereich";
  const showLoginLink =
    inWorkshop && !isAuthenticated && location.pathname !== "/werkstatt/login";

  return (
    <header className="navbar">
      <div className="container navbar__inner">
        <div className="navbar__brand">
          <NavLink to="/" className="navbar__wordmark">
            Werkstattportal
          </NavLink>
          <span className="navbar__section">{sectionLabel}</span>
        </div>

        <div className="navbar__spacer" />

        {inWorkshop && isAuthenticated && employee ? (
          <div className="navbar__account">
            <span className="navbar__employee">{employee.name}</span>
            <button type="button" className="btn btn--ghost" onClick={signOut}>
              Abmelden
            </button>
          </div>
        ) : null}

        <nav aria-label="Hauptnavigation">
          <ul
            id="primary-navigation"
            className="navbar__links"
            data-open={open}
          >
            {links.map((link) => (
              <li key={link.to}>
                <NavLink
                  to={link.to}
                  className={linkClass}
                  end={link.to === "/"}
                  onClick={() => setOpen(false)}
                >
                  {link.label}
                </NavLink>
              </li>
            ))}
            {showLoginLink ? (
              <li>
                <NavLink
                  to="/werkstatt/login"
                  className={linkClass}
                  onClick={() => setOpen(false)}
                >
                  Anmelden
                </NavLink>
              </li>
            ) : null}
          </ul>
        </nav>

        <button
          type="button"
          className="navbar__toggle"
          aria-expanded={open}
          aria-controls="primary-navigation"
          aria-label="Menü"
          onClick={() => setOpen((value) => !value)}
        >
          Menü
        </button>
      </div>
    </header>
  );
}
