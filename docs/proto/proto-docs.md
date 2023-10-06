<!DOCTYPE html>

<html>
  <head>
    <title>Protocol Documentation</title>
    <meta charset="UTF-8">
    <link rel="stylesheet" type="text/css" href="https://fonts.googleapis.com/css?family=Ubuntu:400,700,400italic"/>
    <style>
      body {
        width: 60em;
        margin: 1em auto;
        color: #222;
        font-family: "Ubuntu", sans-serif;
        padding-bottom: 4em;
      }

      h1 {
        font-weight: normal;
        border-bottom: 1px solid #aaa;
        padding-bottom: 0.5ex;
      }

      h2 {
        border-bottom: 1px solid #aaa;
        padding-bottom: 0.5ex;
        margin: 1.5em 0;
      }

      h3 {
        font-weight: normal;
        border-bottom: 1px solid #aaa;
        padding-bottom: 0.5ex;
      }

      a {
        text-decoration: none;
        color: #567e25;
      }

      table {
        width: 100%;
        font-size: 80%;
        border-collapse: collapse;
      }

      thead {
        font-weight: 700;
        background-color: #dcdcdc;
      }

      tbody tr:nth-child(even) {
        background-color: #fbfbfb;
      }

      td {
        border: 1px solid #ccc;
        padding: 0.5ex 2ex;
      }

      td p {
        text-indent: 1em;
        margin: 0;
      }

      td p:nth-child(1) {
        text-indent: 0;  
      }

       
      .field-table td:nth-child(1) {  
        width: 10em;
      }
      .field-table td:nth-child(2) {  
        width: 10em;
      }
      .field-table td:nth-child(3) {  
        width: 6em;
      }
      .field-table td:nth-child(4) {  
        width: auto;
      }

       
      .extension-table td:nth-child(1) {  
        width: 10em;
      }
      .extension-table td:nth-child(2) {  
        width: 10em;
      }
      .extension-table td:nth-child(3) {  
        width: 10em;
      }
      .extension-table td:nth-child(4) {  
        width: 5em;
      }
      .extension-table td:nth-child(5) {  
        width: auto;
      }

       
      .enum-table td:nth-child(1) {  
        width: 10em;
      }
      .enum-table td:nth-child(2) {  
        width: 10em;
      }
      .enum-table td:nth-child(3) {  
        width: auto;
      }

       
      .scalar-value-types-table tr {
        height: 3em;
      }

       
      #toc-container ul {
        list-style-type: none;
        padding-left: 1em;
        line-height: 180%;
        margin: 0;
      }
      #toc > li > a {
        font-weight: bold;
      }

       
      .file-heading {
        width: 100%;
        display: table;
        border-bottom: 1px solid #aaa;
        margin: 4em 0 1.5em 0;
      }
      .file-heading h2 {
        border: none;
        display: table-cell;
      }
      .file-heading a {
        text-align: right;
        display: table-cell;
      }

       
      .badge {
        width: 1.6em;
        height: 1.6em;
        display: inline-block;

        line-height: 1.6em;
        text-align: center;
        font-weight: bold;
        font-size: 60%;

        color: #89ba48;
        background-color: #dff0c8;

        margin: 0.5ex 1em 0.5ex -1em;
        border: 1px solid #fbfbfb;
        border-radius: 1ex;
      }
    </style>

    
    <link rel="stylesheet" type="text/css" href="stylesheet.css"/>
  </head>

  <body>

    <h1 id="title">Protocol Documentation</h1>

    <h2>Table of Contents</h2>

    <div id="toc-container">
      <ul id="toc">
        
          
          <li>
            <a href="#bze%2fburner%2fv1%2fburn_coins_proposal.proto">bze/burner/v1/burn_coins_proposal.proto</a>
            <ul>
              
                <li>
                  <a href="#bze.burner.v1.BurnCoinsProposal"><span class="badge">M</span>BurnCoinsProposal</a>
                </li>
              
              
              
              
            </ul>
          </li>
        
          
          <li>
            <a href="#bze%2fburner%2fv1%2fburned_coins.proto">bze/burner/v1/burned_coins.proto</a>
            <ul>
              
                <li>
                  <a href="#bze.burner.v1.BurnedCoins"><span class="badge">M</span>BurnedCoins</a>
                </li>
              
              
              
              
            </ul>
          </li>
        
          
          <li>
            <a href="#bze%2fburner%2fv1%2fevents.proto">bze/burner/v1/events.proto</a>
            <ul>
              
                <li>
                  <a href="#bze.burner.v1.CoinsBurnedEvent"><span class="badge">M</span>CoinsBurnedEvent</a>
                </li>
              
                <li>
                  <a href="#bze.burner.v1.FundBurnerEvent"><span class="badge">M</span>FundBurnerEvent</a>
                </li>
              
              
              
              
            </ul>
          </li>
        
          
          <li>
            <a href="#bze%2fburner%2fv1%2fparams.proto">bze/burner/v1/params.proto</a>
            <ul>
              
                <li>
                  <a href="#bze.burner.v1.Params"><span class="badge">M</span>Params</a>
                </li>
              
              
              
              
            </ul>
          </li>
        
          
          <li>
            <a href="#bze%2fburner%2fv1%2fgenesis.proto">bze/burner/v1/genesis.proto</a>
            <ul>
              
                <li>
                  <a href="#bze.burner.v1.GenesisState"><span class="badge">M</span>GenesisState</a>
                </li>
              
              
              
              
            </ul>
          </li>
        
          
          <li>
            <a href="#bze%2fburner%2fv1%2fquery.proto">bze/burner/v1/query.proto</a>
            <ul>
              
                <li>
                  <a href="#bze.burner.v1.QueryAllBurnedCoinsRequest"><span class="badge">M</span>QueryAllBurnedCoinsRequest</a>
                </li>
              
                <li>
                  <a href="#bze.burner.v1.QueryAllBurnedCoinsResponse"><span class="badge">M</span>QueryAllBurnedCoinsResponse</a>
                </li>
              
                <li>
                  <a href="#bze.burner.v1.QueryParamsRequest"><span class="badge">M</span>QueryParamsRequest</a>
                </li>
              
                <li>
                  <a href="#bze.burner.v1.QueryParamsResponse"><span class="badge">M</span>QueryParamsResponse</a>
                </li>
              
              
              
              
                <li>
                  <a href="#bze.burner.v1.Query"><span class="badge">S</span>Query</a>
                </li>
              
            </ul>
          </li>
        
          
          <li>
            <a href="#bze%2fburner%2fv1%2ftx.proto">bze/burner/v1/tx.proto</a>
            <ul>
              
                <li>
                  <a href="#bze.burner.v1.MsgFundBurner"><span class="badge">M</span>MsgFundBurner</a>
                </li>
              
                <li>
                  <a href="#bze.burner.v1.MsgFundBurnerResponse"><span class="badge">M</span>MsgFundBurnerResponse</a>
                </li>
              
              
              
              
                <li>
                  <a href="#bze.burner.v1.Msg"><span class="badge">S</span>Msg</a>
                </li>
              
            </ul>
          </li>
        
        <li><a href="#scalar-value-types">Scalar Value Types</a></li>
      </ul>
    </div>

    
      
      <div class="file-heading">
        <h2 id="bze/burner/v1/burn_coins_proposal.proto">bze/burner/v1/burn_coins_proposal.proto</h2><a href="#title">Top</a>
      </div>
      <p></p>

      
        <h3 id="bze.burner.v1.BurnCoinsProposal">BurnCoinsProposal</h3>
        <p></p>

        
          <table class="field-table">
            <thead>
              <tr><td>Field</td><td>Type</td><td>Label</td><td>Description</td></tr>
            </thead>
            <tbody>
              
                <tr>
                  <td>title</td>
                  <td><a href="#string">string</a></td>
                  <td></td>
                  <td><p> </p></td>
                </tr>
              
                <tr>
                  <td>description</td>
                  <td><a href="#string">string</a></td>
                  <td></td>
                  <td><p> </p></td>
                </tr>
              
            </tbody>
          </table>

          

        
      

      

      

      
    
      
      <div class="file-heading">
        <h2 id="bze/burner/v1/burned_coins.proto">bze/burner/v1/burned_coins.proto</h2><a href="#title">Top</a>
      </div>
      <p></p>

      
        <h3 id="bze.burner.v1.BurnedCoins">BurnedCoins</h3>
        <p></p>

        
          <table class="field-table">
            <thead>
              <tr><td>Field</td><td>Type</td><td>Label</td><td>Description</td></tr>
            </thead>
            <tbody>
              
                <tr>
                  <td>burned</td>
                  <td><a href="#string">string</a></td>
                  <td></td>
                  <td><p> </p></td>
                </tr>
              
                <tr>
                  <td>height</td>
                  <td><a href="#string">string</a></td>
                  <td></td>
                  <td><p> </p></td>
                </tr>
              
            </tbody>
          </table>

          

        
      

      

      

      
    
      
      <div class="file-heading">
        <h2 id="bze/burner/v1/events.proto">bze/burner/v1/events.proto</h2><a href="#title">Top</a>
      </div>
      <p></p>

      
        <h3 id="bze.burner.v1.CoinsBurnedEvent">CoinsBurnedEvent</h3>
        <p></p>

        
          <table class="field-table">
            <thead>
              <tr><td>Field</td><td>Type</td><td>Label</td><td>Description</td></tr>
            </thead>
            <tbody>
              
                <tr>
                  <td>burned</td>
                  <td><a href="#string">string</a></td>
                  <td></td>
                  <td><p> </p></td>
                </tr>
              
            </tbody>
          </table>

          

        
      
        <h3 id="bze.burner.v1.FundBurnerEvent">FundBurnerEvent</h3>
        <p></p>

        
          <table class="field-table">
            <thead>
              <tr><td>Field</td><td>Type</td><td>Label</td><td>Description</td></tr>
            </thead>
            <tbody>
              
                <tr>
                  <td>from</td>
                  <td><a href="#string">string</a></td>
                  <td></td>
                  <td><p> </p></td>
                </tr>
              
                <tr>
                  <td>amount</td>
                  <td><a href="#string">string</a></td>
                  <td></td>
                  <td><p> </p></td>
                </tr>
              
            </tbody>
          </table>

          

        
      

      

      

      
    
      
      <div class="file-heading">
        <h2 id="bze/burner/v1/params.proto">bze/burner/v1/params.proto</h2><a href="#title">Top</a>
      </div>
      <p></p>

      
        <h3 id="bze.burner.v1.Params">Params</h3>
        <p>Params defines the parameters for the module.</p>

        

        
      

      

      

      
    
      
      <div class="file-heading">
        <h2 id="bze/burner/v1/genesis.proto">bze/burner/v1/genesis.proto</h2><a href="#title">Top</a>
      </div>
      <p></p>

      
        <h3 id="bze.burner.v1.GenesisState">GenesisState</h3>
        <p>GenesisState defines the burner module's genesis state.</p>

        
          <table class="field-table">
            <thead>
              <tr><td>Field</td><td>Type</td><td>Label</td><td>Description</td></tr>
            </thead>
            <tbody>
              
                <tr>
                  <td>params</td>
                  <td><a href="#bze.burner.v1.Params">Params</a></td>
                  <td></td>
                  <td><p> </p></td>
                </tr>
              
                <tr>
                  <td>burned_coins_list</td>
                  <td><a href="#bze.burner.v1.BurnedCoins">BurnedCoins</a></td>
                  <td>repeated</td>
                  <td><p>this line is used by starport scaffolding # genesis/proto/state </p></td>
                </tr>
              
            </tbody>
          </table>

          

        
      

      

      

      
    
      
      <div class="file-heading">
        <h2 id="bze/burner/v1/query.proto">bze/burner/v1/query.proto</h2><a href="#title">Top</a>
      </div>
      <p></p>

      
        <h3 id="bze.burner.v1.QueryAllBurnedCoinsRequest">QueryAllBurnedCoinsRequest</h3>
        <p></p>

        
          <table class="field-table">
            <thead>
              <tr><td>Field</td><td>Type</td><td>Label</td><td>Description</td></tr>
            </thead>
            <tbody>
              
                <tr>
                  <td>pagination</td>
                  <td><a href="#cosmos.base.query.v1beta1.PageRequest">cosmos.base.query.v1beta1.PageRequest</a></td>
                  <td></td>
                  <td><p> </p></td>
                </tr>
              
            </tbody>
          </table>

          

        
      
        <h3 id="bze.burner.v1.QueryAllBurnedCoinsResponse">QueryAllBurnedCoinsResponse</h3>
        <p></p>

        
          <table class="field-table">
            <thead>
              <tr><td>Field</td><td>Type</td><td>Label</td><td>Description</td></tr>
            </thead>
            <tbody>
              
                <tr>
                  <td>burnedCoins</td>
                  <td><a href="#bze.burner.v1.BurnedCoins">BurnedCoins</a></td>
                  <td>repeated</td>
                  <td><p> </p></td>
                </tr>
              
                <tr>
                  <td>pagination</td>
                  <td><a href="#cosmos.base.query.v1beta1.PageResponse">cosmos.base.query.v1beta1.PageResponse</a></td>
                  <td></td>
                  <td><p> </p></td>
                </tr>
              
            </tbody>
          </table>

          

        
      
        <h3 id="bze.burner.v1.QueryParamsRequest">QueryParamsRequest</h3>
        <p>QueryParamsRequest is request type for the Query/Params RPC method.</p>

        

        
      
        <h3 id="bze.burner.v1.QueryParamsResponse">QueryParamsResponse</h3>
        <p>QueryParamsResponse is response type for the Query/Params RPC method.</p>

        
          <table class="field-table">
            <thead>
              <tr><td>Field</td><td>Type</td><td>Label</td><td>Description</td></tr>
            </thead>
            <tbody>
              
                <tr>
                  <td>params</td>
                  <td><a href="#bze.burner.v1.Params">Params</a></td>
                  <td></td>
                  <td><p>params holds all the parameters of this module. </p></td>
                </tr>
              
            </tbody>
          </table>

          

        
      

      

      

      
        <h3 id="bze.burner.v1.Query">Query</h3>
        <p>Query defines the gRPC querier service.</p>
        <table class="enum-table">
          <thead>
            <tr><td>Method Name</td><td>Request Type</td><td>Response Type</td><td>Description</td></tr>
          </thead>
          <tbody>
            
              <tr>
                <td>Params</td>
                <td><a href="#bze.burner.v1.QueryParamsRequest">QueryParamsRequest</a></td>
                <td><a href="#bze.burner.v1.QueryParamsResponse">QueryParamsResponse</a></td>
                <td><p>Parameters queries the parameters of the module.</p></td>
              </tr>
            
              <tr>
                <td>AllBurnedCoins</td>
                <td><a href="#bze.burner.v1.QueryAllBurnedCoinsRequest">QueryAllBurnedCoinsRequest</a></td>
                <td><a href="#bze.burner.v1.QueryAllBurnedCoinsResponse">QueryAllBurnedCoinsResponse</a></td>
                <td><p></p></td>
              </tr>
            
          </tbody>
        </table>

        
          
          
          <h4>Methods with HTTP bindings</h4>
          <table>
            <thead>
              <tr>
                <td>Method Name</td>
                <td>Method</td>
                <td>Pattern</td>
                <td>Body</td>
              </tr>
            </thead>
            <tbody>
            
              
              
              <tr>
                <td>Params</td>
                <td>GET</td>
                <td>/bze/burner/v1/params</td>
                <td></td>
              </tr>
              
            
              
              
              <tr>
                <td>AllBurnedCoins</td>
                <td>GET</td>
                <td>/bze/burner/v1/all_burned_coins</td>
                <td></td>
              </tr>
              
            
            </tbody>
          </table>
          
        
    
      
      <div class="file-heading">
        <h2 id="bze/burner/v1/tx.proto">bze/burner/v1/tx.proto</h2><a href="#title">Top</a>
      </div>
      <p></p>

      
        <h3 id="bze.burner.v1.MsgFundBurner">MsgFundBurner</h3>
        <p></p>

        
          <table class="field-table">
            <thead>
              <tr><td>Field</td><td>Type</td><td>Label</td><td>Description</td></tr>
            </thead>
            <tbody>
              
                <tr>
                  <td>creator</td>
                  <td><a href="#string">string</a></td>
                  <td></td>
                  <td><p> </p></td>
                </tr>
              
                <tr>
                  <td>amount</td>
                  <td><a href="#string">string</a></td>
                  <td></td>
                  <td><p> </p></td>
                </tr>
              
            </tbody>
          </table>

          

        
      
        <h3 id="bze.burner.v1.MsgFundBurnerResponse">MsgFundBurnerResponse</h3>
        <p></p>

        

        
      

      

      

      
        <h3 id="bze.burner.v1.Msg">Msg</h3>
        <p>Msg defines the Msg service.</p>
        <table class="enum-table">
          <thead>
            <tr><td>Method Name</td><td>Request Type</td><td>Response Type</td><td>Description</td></tr>
          </thead>
          <tbody>
            
              <tr>
                <td>FundBurner</td>
                <td><a href="#bze.burner.v1.MsgFundBurner">MsgFundBurner</a></td>
                <td><a href="#bze.burner.v1.MsgFundBurnerResponse">MsgFundBurnerResponse</a></td>
                <td><p>this line is used by starport scaffolding # proto/tx/rpc</p></td>
              </tr>
            
          </tbody>
        </table>

        
    

    <h2 id="scalar-value-types">Scalar Value Types</h2>
    <table class="scalar-value-types-table">
      <thead>
        <tr><td>.proto Type</td><td>Notes</td><td>C++</td><td>Java</td><td>Python</td><td>Go</td><td>C#</td><td>PHP</td><td>Ruby</td></tr>
      </thead>
      <tbody>
        
          <tr id="double">
            <td>double</td>
            <td></td>
            <td>double</td>
            <td>double</td>
            <td>float</td>
            <td>float64</td>
            <td>double</td>
            <td>float</td>
            <td>Float</td>
          </tr>
        
          <tr id="float">
            <td>float</td>
            <td></td>
            <td>float</td>
            <td>float</td>
            <td>float</td>
            <td>float32</td>
            <td>float</td>
            <td>float</td>
            <td>Float</td>
          </tr>
        
          <tr id="int32">
            <td>int32</td>
            <td>Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint32 instead.</td>
            <td>int32</td>
            <td>int</td>
            <td>int</td>
            <td>int32</td>
            <td>int</td>
            <td>integer</td>
            <td>Bignum or Fixnum (as required)</td>
          </tr>
        
          <tr id="int64">
            <td>int64</td>
            <td>Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint64 instead.</td>
            <td>int64</td>
            <td>long</td>
            <td>int/long</td>
            <td>int64</td>
            <td>long</td>
            <td>integer/string</td>
            <td>Bignum</td>
          </tr>
        
          <tr id="uint32">
            <td>uint32</td>
            <td>Uses variable-length encoding.</td>
            <td>uint32</td>
            <td>int</td>
            <td>int/long</td>
            <td>uint32</td>
            <td>uint</td>
            <td>integer</td>
            <td>Bignum or Fixnum (as required)</td>
          </tr>
        
          <tr id="uint64">
            <td>uint64</td>
            <td>Uses variable-length encoding.</td>
            <td>uint64</td>
            <td>long</td>
            <td>int/long</td>
            <td>uint64</td>
            <td>ulong</td>
            <td>integer/string</td>
            <td>Bignum or Fixnum (as required)</td>
          </tr>
        
          <tr id="sint32">
            <td>sint32</td>
            <td>Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int32s.</td>
            <td>int32</td>
            <td>int</td>
            <td>int</td>
            <td>int32</td>
            <td>int</td>
            <td>integer</td>
            <td>Bignum or Fixnum (as required)</td>
          </tr>
        
          <tr id="sint64">
            <td>sint64</td>
            <td>Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int64s.</td>
            <td>int64</td>
            <td>long</td>
            <td>int/long</td>
            <td>int64</td>
            <td>long</td>
            <td>integer/string</td>
            <td>Bignum</td>
          </tr>
        
          <tr id="fixed32">
            <td>fixed32</td>
            <td>Always four bytes. More efficient than uint32 if values are often greater than 2^28.</td>
            <td>uint32</td>
            <td>int</td>
            <td>int</td>
            <td>uint32</td>
            <td>uint</td>
            <td>integer</td>
            <td>Bignum or Fixnum (as required)</td>
          </tr>
        
          <tr id="fixed64">
            <td>fixed64</td>
            <td>Always eight bytes. More efficient than uint64 if values are often greater than 2^56.</td>
            <td>uint64</td>
            <td>long</td>
            <td>int/long</td>
            <td>uint64</td>
            <td>ulong</td>
            <td>integer/string</td>
            <td>Bignum</td>
          </tr>
        
          <tr id="sfixed32">
            <td>sfixed32</td>
            <td>Always four bytes.</td>
            <td>int32</td>
            <td>int</td>
            <td>int</td>
            <td>int32</td>
            <td>int</td>
            <td>integer</td>
            <td>Bignum or Fixnum (as required)</td>
          </tr>
        
          <tr id="sfixed64">
            <td>sfixed64</td>
            <td>Always eight bytes.</td>
            <td>int64</td>
            <td>long</td>
            <td>int/long</td>
            <td>int64</td>
            <td>long</td>
            <td>integer/string</td>
            <td>Bignum</td>
          </tr>
        
          <tr id="bool">
            <td>bool</td>
            <td></td>
            <td>bool</td>
            <td>boolean</td>
            <td>boolean</td>
            <td>bool</td>
            <td>bool</td>
            <td>boolean</td>
            <td>TrueClass/FalseClass</td>
          </tr>
        
          <tr id="string">
            <td>string</td>
            <td>A string must always contain UTF-8 encoded or 7-bit ASCII text.</td>
            <td>string</td>
            <td>String</td>
            <td>str/unicode</td>
            <td>string</td>
            <td>string</td>
            <td>string</td>
            <td>String (UTF-8)</td>
          </tr>
        
          <tr id="bytes">
            <td>bytes</td>
            <td>May contain any arbitrary sequence of bytes.</td>
            <td>string</td>
            <td>ByteString</td>
            <td>str</td>
            <td>[]byte</td>
            <td>ByteString</td>
            <td>string</td>
            <td>String (ASCII-8BIT)</td>
          </tr>
        
      </tbody>
    </table>
  </body>
</html>

