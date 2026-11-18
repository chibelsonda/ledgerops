import { Menu, MenuButton, MenuItem, MenuItems } from '@headlessui/react'
import { useQuery } from '@tanstack/react-query'
import { BriefcaseBusiness, ChartNoAxesCombined, ChevronDown, CircleDollarSign, FileText, LogOut, MailCheck, RefreshCw, ScanSearch, Users } from 'lucide-react'
import { NavLink, Outlet } from 'react-router-dom'
import { dashboardApi } from '../features/dashboard/api'
import { useAuth } from '../features/auth/AuthProvider'
import type { Role } from '../features/auth/types'
import { titleCase } from '../shared/lib/format'

const navigation = [
  { to: '/', label: 'Overview', icon: ChartNoAxesCombined, end: true, roles: ['admin', 'controller', 'reviewer'] as Role[] },
  { to: '/invoices', label: 'Invoices', icon: FileText, roles: ['admin', 'controller', 'reviewer'] as Role[] },
  { to: '/clients', label: 'Clients', icon: Users, roles: ['admin', 'controller', 'reviewer'] as Role[] },
  { to: '/projects', label: 'Projects', icon: BriefcaseBusiness, roles: ['admin', 'controller', 'reviewer'] as Role[] },
  { to: '/collections', label: 'Collections', icon: MailCheck, roles: ['admin', 'controller', 'reviewer'] as Role[] },
  { to: '/review', label: 'Review', icon: ScanSearch, roles: ['admin', 'controller', 'reviewer'] as Role[] },
  { to: '/sync', label: 'Data sync', icon: RefreshCw, roles: ['admin', 'controller'] as Role[] },
]

export function AppShell() {
  const { session, logout } = useAuth()
  const { data } = useQuery({ queryKey: ['dashboard'], queryFn: dashboardApi.get })
  if (!session) return null
  const initials = session.user.displayName.split(/\s+/).map((name) => name[0]).join('').slice(0, 2).toUpperCase()
  return (
    <div className="app-shell">
      <aside className="sidebar">
        <div className="brand"><span className="brand-mark" aria-hidden="true"><CircleDollarSign size={22} /></span><span>LedgerOps</span></div>
        <nav aria-label="Primary navigation">
          {navigation.filter((item) => item.roles.includes(session.user.role)).map(({ to, label, icon: Icon, end }) => (
            <NavLink key={to} to={to} end={end} className={({ isActive }) => isActive ? 'nav-link active' : 'nav-link'}>
              <Icon size={18} strokeWidth={1.8} /><span>{label}</span>
              {label === 'Review' && data?.metrics.openReviewCount ? <span className="nav-count">{data.metrics.openReviewCount}</span> : null}
            </NavLink>
          ))}
        </nav>
      </aside>
      <main className="main-content">
        <header className="main-content-header">
          <div className="main-content-header-inner">
            <strong>Admin Panel</strong>
            <Menu as="div" className="user-menu">
              <MenuButton className="user-menu-trigger">
                <span className="avatar">{initials}</span>
                <span className="user-menu-copy"><strong>{session.user.displayName}</strong><small>{titleCase(session.user.role)}</small></span>
                <ChevronDown size={15} aria-hidden="true" />
              </MenuButton>
              <MenuItems anchor={{ to: 'bottom end', gap: 8 }} className="user-menu-dropdown">
                <div className="user-menu-details">
                  <strong>{session.user.displayName}</strong>
                  <span>{session.user.email}</span>
                </div>
                <MenuItem as="button" type="button" className="user-menu-signout" onClick={() => void logout()}>
                  <LogOut size={16} aria-hidden="true" /> Sign out
                </MenuItem>
              </MenuItems>
            </Menu>
          </div>
        </header>
        <Outlet />
      </main>
    </div>
  )
}
