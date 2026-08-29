// Minimal, dependency-free collapse/expand for the per-resource component tables.
(function () {
  document.addEventListener("click", function (e) {
    var btn = e.target;
    if (!btn || btn.className !== "toggle") return;
    var inner = btn.parentNode.querySelector("table.inner");
    if (inner) inner.hidden = !inner.hidden;
  });
})();
