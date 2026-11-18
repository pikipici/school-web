"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { NAV_ITEMS } from "@/lib/site";
import { activePublicNav } from "@/lib/public-content";

export function PublicNavigation() {
 const active = activePublicNav(usePathname());
 return <nav className="public-navigation" aria-label="Menu utama"><ul className="nav-list">{NAV_ITEMS.map(item => <li key={item.href}>
  <Link className="nav-link" href={item.href} aria-current={active === item.href ? "page" : undefined}>{item.label}</Link>
 </li>)}</ul></nav>;
}
