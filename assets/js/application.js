require("expose-loader?exposes=$,jQuery!jquery");
require("bootstrap/dist/js/bootstrap.bundle.js");
require("@fortawesome/fontawesome-free/js/all.js");
require("jquery-ujs/src/rails.js");

$(() => {
  $("[data-pix-key-form]").on("submit", function () {
    const accountId = $(this).find("[name='account_id']").val();
    this.action = `/accounts/${encodeURIComponent(accountId)}/pix-keys`;
  });

  $("[data-pix-lookup-form]").on("submit", function (event) {
    event.preventDefault();
    const key = $(this).find("[name='key']").val();
    if (key) {
      window.location.href = `/pix-keys/${encodeURIComponent(key)}`;
    }
  });
});
