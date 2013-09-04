<?php
  if ($_POST) {
    $thing = preg_replace('/\n/', '', $_POST['thing']);
    $t = date('Y m d H i s');
    $tl = fopen('/tmp/tl.log', 'a');
    fwrite($tl, "$t $thing\n");
    fclose($tl);

    // Work around chrome bug: force reload
    ?><html><body onload="location=location;"></body></html><?php
    exit;

  }
?><html>
<head><style type="text/css">
  body, form, input { font-size: 48pt }
</style></head>
<body onload="document.getElementById('thing').focus();">
<form action="tl.php" method="post">
  <!-- The submit button is above input field so that it is not obscured
       by the autocomplete dropdown. -->
  <p><input type="submit"></p>
  <p><input type="text" name="thing" id="thing" autofocus></p>
</form>
</body></html>
