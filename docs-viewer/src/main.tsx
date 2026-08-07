import { mountPortal } from "./mount/mount-portal";
import "./styles.css";

const container = document.getElementById("root");

if (!container) {
  throw new Error("マウント先の #root がありません");
}

void mountPortal(container);
