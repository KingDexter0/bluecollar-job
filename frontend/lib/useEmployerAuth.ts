"use client";

import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { getToken } from "@/lib/api";

export function useEmployerAuth(redirectTo = "/employer/login") {
  const router = useRouter();
  const [ready, setReady] = useState(false);

  useEffect(() => {
    if (!getToken()) {
      router.replace(redirectTo);
      return;
    }
    setReady(true);
  }, [redirectTo, router]);

  return ready;
}
