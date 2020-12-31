/**
* @Author:zhoutao
* @Date:2020/12/30 下午3:02
* @Desc:
 */

package client

const debugText = `<html>
	<body>
	<title> rpc services </title>
	{{range .}}
	<hr>
	Service {{.Name}}
	<hr>
		<table>
		<th align=center>Method</th><th align=center>Calls</th>
		{{range $name,$mtype:=.Method}}
			<tr>
			<td align=left font=fixed>{{$name}} error</td>
			<td align=center>{{$mtype.NumCalls}}</td>
			</tr>
		{{end}}
		</table>
	{{end}}
	</body>
</html>
`
