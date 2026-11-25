package service
import ("testing";"reflect";"context";"github.com/arashrasoulzadeh/eye/internal/model")
func TestDimensionValues(t *testing.T){
 if got:=dimensionValues([]any{"a","b","a"},true);!reflect.DeepEqual(got,[]string{"a","b"}){t.Fatal(got)}
 if got:=dimensionValues(nil,true);!reflect.DeepEqual(got,[]string{"(missing)"}){t.Fatal(got)}
}

func TestNestedDimensionPaths(t *testing.T){
 got:=flattenDimensions(map[string]any{"device":map[string]any{"os":"Android"},"a/b":42.0,"a~b":true},"",0)
 if got["/device/os"]!="Android"||got["/a~1b"]!=42.0||got["/a~0b"]!=true{t.Fatal(got)}
}
func TestDimensionFilters(t *testing.T){
 dims:=map[string]any{"tags":[]any{"Math","Science"},"score":0.0}
 for _,c:=range []struct{f DimensionFilter;want bool}{
 {DimensionFilter{Key:"tags",Mode:"contains",Value:"math"},true},
 {DimensionFilter{Key:"tags",Mode:"equal",Value:"Science"},true},
 {DimensionFilter{Key:"tags",Mode:"not_equal",Value:"Science"},false},
 {DimensionFilter{Key:"score",Mode:"exists"},true},
 {DimensionFilter{Key:"unknown",Mode:"missing"},true},
 }{if got:=matchDimensionFilter(dims,c.f);got!=c.want{t.Errorf("%+v: %v",c.f,got)}}
}

func TestDimensionPairStats(t *testing.T) {
 var positive,negative,constant,small dimensionPairStats
 for i:=0;i<100;i++{x:=float64(i)+1e9;positive.add(x,2*x);negative.add(x,-x);constant.add(x,1);if i<10{small.add(x,x)}}
 if r,ok:=positive.coefficient();!ok||r<0.999999{t.Fatalf("positive: %v %v",r,ok)}
 if r,ok:=negative.coefficient();!ok||r> -0.999999{t.Fatalf("negative: %v %v",r,ok)}
 if _,ok:=constant.coefficient();ok{t.Fatal("constant series must not correlate")}
 if _,ok:=small.coefficient();ok{t.Fatal("small sample must not correlate")}
}

func TestDimensionSummaryScope(t *testing.T) {
 db:=newTestDB_dashboard_cache_test(t)
 events:=[]model.Event{
 {EventID:"a",TenantID:"t1",UserID:"u1",SessionID:"s1",EventName:"click",ClientTS:100},
 {EventID:"b",TenantID:"t1",UserID:"u1",SessionID:"s2",EventName:"click",ClientTS:101},
 {EventID:"c",TenantID:"t1",SessionID:"s2",EventName:"click",ClientTS:102},
 {EventID:"d",TenantID:"t1",UserID:"u2",EventName:"view",ClientTS:103},
 {EventID:"e",TenantID:"t2",UserID:"u3",SessionID:"s3",EventName:"click",ClientTS:104},
 {EventID:"f",TenantID:"t1",UserID:"u4",SessionID:"s4",EventName:"click",ClientTS:99},
 }
 if err:=db.Create(&events).Error;err!=nil{t.Fatal(err)}
 q:=&QueryService{DB:db}
 got,err:=q.AnalyzeDimensions(context.Background(),DimensionInput{Tenant:"t1",From:100,To:104,Types:[]string{"click"},Operation:"count",Summary:true})
 if err!=nil{t.Fatal(err)}
 if got.Matched!=3||got.Users!=1||got.Sessions!=2||got.IdentifiedEvents!=2{t.Fatalf("unexpected summary: %+v",got)}
}

func TestCombinedDimensionSummaryTrend(t *testing.T) {
 db:=newTestDB_dashboard_cache_test(t)
 events:=[]model.Event{
 {EventID:"a",TenantID:"t1",UserID:"u1",SessionID:"s1",EventName:"click",ClientTS:100},
 {EventID:"b",TenantID:"t1",UserID:"u1",SessionID:"s2",EventName:"click",ClientTS:3600000},
 {EventID:"c",TenantID:"t1",EventName:"click",ClientTS:3600001},
 {EventID:"d",TenantID:"t2",EventName:"click",ClientTS:3600001},
 {EventID:"e",TenantID:"t1",EventName:"view",ClientTS:3600001},
 {EventID:"f",TenantID:"t1",EventName:"click",ClientTS:7200000},
 }
 if err:=db.Create(&events).Error;err!=nil{t.Fatal(err)}
 q:=&QueryService{DB:db}
 got,err:=q.AnalyzeDimensions(context.Background(),DimensionInput{Tenant:"t1",From:0,To:7199999,Types:[]string{"click"},Operation:"count",Summary:true,IncludeTrend:true,TimeBucket:"hour"})
 if err!=nil{t.Fatal(err)}
 if got.Matched!=3||got.Users!=1||got.Sessions!=2||got.IdentifiedEvents!=2||got.Execution!="database"{t.Fatalf("unexpected summary: %+v",got)}
 if len(got.Rows)!=2||got.Rows[0].Group!="1970-01-01 00:00"||got.Rows[0].Count!=1||got.Rows[1].Group!="1970-01-01 01:00"||got.Rows[1].Count!=2{t.Fatalf("unexpected buckets: %+v",got.Rows)}
}
